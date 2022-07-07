package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	eiop "github.com/njones/socketio/engineio/protocol"
	"golang.org/x/sync/errgroup"
	ws "nhooyr.io/websocket"
)

type ctky string

const serverSetupComplete ctky = "server_setup_complete"

type WebsocketTransport struct {
	*Transport

	origin []string

	conn    *ws.Conn
	PingMsg string
}

func NewWebsocketTransport(chanBuf int) func(SessionID, Codec) Transporter {
	return func(id SessionID, codec Codec) Transporter {
		{
			t := &WebsocketTransport{
				Transport: &Transport{
					id:      id,
					name:    "websocket",
					codec:   codec,
					send:    make(chan eiop.Packet, chanBuf),
					receive: make(chan eiop.Packet, chanBuf),
				},
				origin:  []string{"*"},
				PingMsg: "probe",
			}

			return t
		}
	}
}

func (t *WebsocketTransport) Run(w http.ResponseWriter, r *http.Request, opts ...Option) (err error) {
	for _, opt := range opts {
		opt(t)
	}

	ctx, cancel := context.WithCancel(r.Context())
	t.Transport.shutdown = func() { cancel() }

	t.conn, err = ws.Accept(w, r, &ws.AcceptOptions{
		OriginPatterns: t.origin,
	})
	if err != nil {
		cancel()
		return err
	}

	// A context value can be passed in to allow the a server to be setup before the
	// probe is attempted, this is good for testing. If the context key is not here
	// then nothing happens and it's skipped.
	if complete := ctx.Value(serverSetupComplete).(*sync.WaitGroup); complete != nil {
		complete.Wait()
	}

	if err := t.probe(w, r.WithContext(ctx)); err != nil {
		cancel()
		return err
	}

	grp, ctx := errgroup.WithContext(ctx)
	grp.Go(func() error { return t.incoming(ctx) })
	grp.Go(func() error { return t.outgoing(r.WithContext(ctx)) })

	return grp.Wait()
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

	// Reveive the Pong
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
	for {
		select {
		case <-ctx.Done():
			return nil
		case packet := <-t.receive:
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

				if err = t.codec.PacketEncoder.To(cw).WritePacket(packet); err != nil {
					return err
				}
				cw.Close()
			}
		}
	}
}

func (t *WebsocketTransport) outgoing(r *http.Request) error {
	ctx := r.Context()
	enc := t.codec.PacketEncoder
	dec := t.codec.PacketDecoder
	for {

		// - /* blocking */ read a packet off the wire...
		mt, cr, err := t.conn.Reader(ctx)
		if err != nil {
			return err
		}

		if mt != ws.MessageText {
			// this is binary data
			t.send <- eiop.Packet{
				T: eiop.BinaryPacket,
				D: cr,
			}
			continue
		}

		var packet eiop.Packet
		if err = dec.From(cr).ReadPacket(&packet); err != nil {
			return err
		}

		switch packet.T {
		case eiop.ClosePacket:
			t.conn.CloseRead(ctx)
			t.conn.Close(ws.StatusNormalClosure, "cross origin WebSocket accepted")

			close(t.send)
			close(t.receive)
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
		case eiop.MessagePacket:
			t.send <- packet
		case eiop.UpgradePacket:
		}
	}
}

func WithPerMessageDeflate(kind HTTPCompressionKind) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case *WebsocketTransport:
			_ = v
		}
	}
}
