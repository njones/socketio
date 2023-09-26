package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	eios "github.com/njones/socketio/engineio/session"
	errg "golang.org/x/sync/errgroup"
	ws "nhooyr.io/websocket"
)

type ctky string

const Websocket Name = "websocket"
const serverSetupComplete ctky = "server_setup_complete"
const defaultPingMsg = "probe"

type WebsocketTransport struct {
	*Transport
	conn *ws.Conn

	origin      []string
	PingMsg     string
	buffered    bool // default: false
	isInitProbe bool
	fnOnUpgrade func() error
	governor    struct {
		sleep   time.Duration
		minTime time.Duration
	}
}

func NewWebsocketTransport(chanBuf int) func(SessionID, Codec) Transporter {
	return func(id SessionID, codec Codec) Transporter {
		{
			t := &WebsocketTransport{
				Transport: &Transport{
					id:       id,
					name:     Websocket,
					codec:    codec,
					send:     make(chan eiop.Packet, chanBuf),
					receive:  make(chan eiop.Packet, chanBuf),
					expireId: make(chan eios.ID),
				},
				origin:  []string{"*"},
				PingMsg: defaultPingMsg,
			}

			return t
		}
	}
}

func (t *WebsocketTransport) With(opts ...Option) {
	for _, opt := range opts {
		opt(t)
	}
}

func (t *WebsocketTransport) InnerTransport() *Transport { return t.Transport }

func (t *WebsocketTransport) Run(w http.ResponseWriter, r *http.Request, opts ...Option) (err error) {
	t.With(opts...)

	t.conn, err = ws.Accept(w, r, &ws.AcceptOptions{
		OriginPatterns: t.origin,
	})
	if err != nil {
		return err
	}

	ctx := r.Context()
	// A context value can be passed in to allow the a server to be setup before the
	// probe is attempted, this is good for testing. If the context key is not here
	// then nothing happens and it's skipped.
	if complete, ok := ctx.Value(serverSetupComplete).(*sync.WaitGroup); ok && complete != nil {
		complete.Wait()
	}

	if t.isInitProbe {
		if err := t.probe(w, r); err != nil {
			return err
		}
	}

	grp, ctx := errg.WithContext(ctx)
	grp.Go(func() error { return t.incoming(ctx) })
	grp.Go(func() error { return t.outgoing(r.WithContext(ctx)) })

	err = grp.Wait()
	t.conn.Close(ws.StatusNormalClosure, "done")
	return err
}

func (t *WebsocketTransport) probe(w http.ResponseWriter, r *http.Request) error {
	type Packet = eiop.Packet

	ctx := r.Context()
	enc := t.codec.PacketEncoder
	dec := t.codec.PacketDecoder

	// Send the Ping...
	wsw, err := t.conn.Writer(ctx, ws.MessageText)
	if err != nil {
		return err
	}

	if err := enc.To(wsw).WritePacket(Packet{T: eiop.PingPacket, D: t.PingMsg}); err != nil {
		return err
	}

	wsw.Close() // done with the connection, must always close.

	// Receive the Pong
	_, wsr, err := t.conn.Reader(ctx)
	if err != nil {
		return err
	}

	var packet Packet
	if err = dec.From(wsr).ReadPacket(&packet); err != nil {
		return err
	}

	if packet.T != eiop.PongPacket {
		return fmt.Errorf("expected pong packet")
	}

	if pingMsg, ok := packet.D.(string); !ok {
		return fmt.Errorf("expected pong word to be a string")
	} else if pingMsg != t.PingMsg {
		return fmt.Errorf("expected pong word is invalid")
	}

	// then we are successful!
	return nil
}

func (t *WebsocketTransport) incoming(ctx context.Context) (err error) {
	var interval, timeout, cancel = make(<-chan time.Time), make(<-chan struct{}), make(<-chan func())
	if fn, ok := ctx.Value(eios.SessionIntervalKey).(eios.IntervalChannel); ok {
		interval = fn()
	}
	if fn, ok := ctx.Value(eios.SessionTimeoutKey).(eios.TimeoutChannel); ok {
		timeout = fn()
	}
	if fn, ok := ctx.Value(eios.SessionCloseChannelKey).(func() <-chan func()); ok {
		cancel = fn()
	}
	extendTimeout, ok := ctx.Value(eios.SessionExtendTimeoutKey).(eios.ExtendTimeoutFunc)
	if !ok {
		extendTimeout = func() {}
	}

	var done func()
	var reason string
	defer func() { t.conn.Close(ws.StatusNormalClosure, reason) }()

	var start = time.Now()
Write:

	for {
		select {
		case stop := <-cancel:
			if stop != nil {
				done = stop
			}
			reason = "stop"
			break Write
		case <-timeout:
			reason = "timeout"
			break Write
		case <-interval:
			reason = "interval"
			cw, err := t.conn.Writer(ctx, ws.MessageText)
			if err != nil {
				if cw != nil {
					cw.Close()
				}
				return err
			}

			if err = t.codec.PacketEncoder.To(cw).WritePacket(eiop.Packet{T: eiop.PingPacket, D: nil}); err != nil {
				cw.Close()
				return err
			}
			cw.Close()
		case packet := <-t.receive:
			reason = "receive"
			extendTimeout()
			if packet.T == eiop.BinaryPacket {
				cw, err := t.conn.Writer(ctx, ws.MessageBinary)
				if err != nil {
					return err
				}

				io.Copy(cw, packet.D.(io.Reader))
				cw.Close()
			} else {

				cw, err := t.conn.Writer(ctx, ws.MessageText)
				if err != nil {
					return err
				}

				if t.governor.minTime > 0 {
					// we need to slow things down sometimes...
					if time.Since(start) < t.governor.minTime {
						time.Sleep(t.governor.sleep)
					}
					start = time.Now()
				}

				err = t.codec.PacketEncoder.To(cw).WritePacket(packet)
				cw.Close()
			}
		}
	}

	select {
	case stop := <-cancel:
		if stop != nil {
			done = stop
		}
		if done != nil {
			defer done()
		}
	default:
	}

	return nil
}

type syncReader struct {
	r io.Reader
	s *sync.WaitGroup
}

func (r syncReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if errors.Is(err, io.EOF) {
		r.s.Done()
	}
	return n, err
}

func (t *WebsocketTransport) outgoing(r *http.Request) (err error) {
	ctx, enc, dec := r.Context(), t.codec.PacketEncoder, t.codec.PacketDecoder
	extendTimeout, ok := ctx.Value(eios.SessionExtendTimeoutKey).(eios.ExtendTimeoutFunc)
	if !ok {
		extendTimeout = func() {}
	}

	var unbuffered = new(sync.WaitGroup)
	defer t.conn.Close(ws.StatusNormalClosure, "read")

	for {
		if !t.buffered {
			unbuffered.Wait()
		}

		// - /* blocking */ -//
		// read a packet off the wire...
		msgType, cr, err := t.conn.Reader(ctx) // this will close when shutdown() is called.
		if err != nil {
			return err
		}

		extendTimeout()
		if msgType != ws.MessageText {
			// this is binary data
			if t.buffered {
				var buf = new(bytes.Buffer)
				_, err := buf.ReadFrom(cr)
				if err != nil {
					return err
				}
				t.send <- eiop.Packet{
					T: eiop.BinaryPacket,
					D: buf,
				}
			} else {
				unbuffered.Add(1)
				t.send <- eiop.Packet{
					T: eiop.BinaryPacket,
					D: syncReader{r: cr, s: unbuffered},
				}
			}
			continue
		}

		var packet eiop.Packet
		if err = dec.From(cr).ReadPacket(&packet); err != nil {
			return err
		}

		switch packet.T {
		case eiop.ClosePacket:
			if done, ok := r.Context().Value(eios.SessionCloseFunctionKey).(func() func()); ok {
				if cleanup := done(); cleanup != nil {
					cleanup()
				}
			}
			t.conn.CloseRead(ctx)
			t.conn.Close(ws.StatusNormalClosure, "cross origin WebSocket accepted")
			return nil
		case eiop.PingPacket:
			cw, err := t.conn.Writer(ctx, ws.MessageText)
			if err != nil {
				return err
			}
			packet.T = eiop.PongPacket
			if err = enc.To(cw).WritePacket(packet); err != nil {
				return err
			}
			cw.Close()
		case eiop.PongPacket:
			continue
		case eiop.MessagePacket:
			t.send <- packet
		case eiop.UpgradePacket:
			if done, ok := r.Context().Value(eios.SessionCloseFunctionKey).(func() func()); ok {
				_ = done() // skip cleanup...
				if t.fnOnUpgrade != nil {
					if err := t.fnOnUpgrade(); err != nil {
						return err
					}
				}
			}

		}
	}
}

func WithPerMessageDeflate(kind HTTPCompressionKind) Option {
	return func(o OptionWith) {
		if v, ok := o.(*WebsocketTransport); ok {
			_ = v
		}
	}
}
