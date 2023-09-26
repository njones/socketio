package transport

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	eios "github.com/njones/socketio/engineio/session"
	"golang.org/x/text/transform"
)

const (
	Polling Name = "polling"
)

type handlerWithError func(http.ResponseWriter, *http.Request) error

type PollingTransport struct {
	*Transport

	sleep time.Duration

	compress func(handlerWithError) handlerWithError
}

func NewPollingTransport(chanBuf int) func(SessionID, Codec) Transporter {
	return func(id SessionID, codec Codec) Transporter {
		t := &PollingTransport{
			Transport: &Transport{
				id:       id,
				name:     Polling,
				codec:    codec,
				send:     make(chan eiop.Packet, chanBuf),
				receive:  make(chan eiop.Packet, chanBuf),
				sendPing: true,
				expireId: make(chan eios.ID),
			},
			compress: func(fn handlerWithError) handlerWithError {
				return func(w http.ResponseWriter, r *http.Request) error {
					return fn(w, r)
				}
			},
			sleep: 25 * time.Millisecond,
		}

		return t
	}
}

func (t *PollingTransport) With(opts ...Option) {
	for _, opt := range opts {
		opt(t)
	}
}

func (t *PollingTransport) InnerTransport() *Transport { return t.Transport }

func (t *PollingTransport) Run(w http.ResponseWriter, r *http.Request, opts ...Option) (err error) {
	t.With(opts...)

	switch r.Method {
	case http.MethodGet:
		return t.compress(jsonp(t.poll))(w, r)
	case http.MethodPost:
		// decompression will happen automatically
		return t.emit(w, r)
	}
	return nil

}

func (t *PollingTransport) Write(w http.ResponseWriter, r *http.Request) error {
	var packets eiop.Payload
	var buffer = &struct {
		use bool
		idx int
	}{}

Write:
	for packet := range t.receive {
		if packet.T == eiop.NoopPacket {
			switch v := packet.D.(type) {
			case WriteClose:
				if !buffer.use {
					buffer.idx = len(packets)
				}
				break Write
			case StartWriteBuffer:
				packets = append(packets, packet)
				if v() {
					buffer.use = true
					buffer.idx = len(packets)
					continue
				}
			}
		}
		packets = append(packets, packet)
	}

	if buffer.use {
		for _, packet := range packets[buffer.idx:] {
			t.receive <- packet
		}
	}

	return t.codec.PayloadEncoder.To(w).WritePayload(packets[:buffer.idx])
}

// longPoll allows a connection for a specified amout of time... then releases a payload
func (t *PollingTransport) poll(w http.ResponseWriter, r *http.Request) (err error) {

	var ctx = r.Context()
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

	var done func()
	var packets eiop.Payload

Write:
	for {
		select {
		case packet := <-t.receive:
			packets = append(packets, packet)
		case stop := <-cancel:
			if stop != nil {
				done = stop
			}
			break Write
		case <-timeout:
			break Write
		case <-interval:
			for i := 0; i < len(t.receive); i++ {
				packet := <-t.receive
				packets = append(packets, packet)
			}
			if len(packets) == 0 && t.sendPing {
				packets = append(packets, eiop.Packet{T: eiop.PingPacket, D: nil})
			}
			break Write
		default:
			time.Sleep(t.sleep) // let other things come in if things are coming quick...
			if len(packets) > 0 && len(t.receive) == 0 {
				break Write
			}
		}
	}

	select {
	case stop := <-cancel:
		if stop != nil {
			done = stop
		}
		err = t.codec.PacketEncoder.To(w).WritePacket(eiop.Packet{T: eiop.NoopPacket})
		if done != nil {
			defer done()
		}
	default:
		if len(packets) > 0 {
			if err := t.codec.PayloadEncoder.To(w).WritePayload(packets); err != nil {
				t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{err}}
				return ErrEncodeFailed.F("polling", err)
			}
		}
	}

	t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{ErrCloseSocket}} // shutdown the HTTP connection
	return err
}

// gather pulls in all of the posts
func (t *PollingTransport) emit(w http.ResponseWriter, r *http.Request) error {

	var payload eiop.Payload
	if err := t.codec.PayloadDecoder.From(r.Body).ReadPayload(&payload); err != nil {
		t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{err}}
		return ErrDecodeFailed.F("polling", err)
	}

Read:
	for _, packet := range payload {
		switch packet.T {
		case eiop.ClosePacket:
			if done, ok := r.Context().Value(eios.SessionCloseFunctionKey).(func() func()); ok {
				if cleanup := done(); cleanup != nil {
					cleanup()
				}
			}
			break Read
		case eiop.PongPacket:
			t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{ErrCloseSocket}}
			return nil
		}
		t.send <- packet
	}

	t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{}} // shutdown the HTTP connection

	return nil
}

type HTTPCompressionKind string

const (
	CompressGZIP HTTPCompressionKind = "gzip"
)

type compressResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (z compressResponseWriter) Write(p []byte) (n int, err error) { return z.Writer.Write(p) }

func WithHTTPCompression(kind HTTPCompressionKind) Option {
	return func(o OptionWith) {
		if v, ok := o.(*PollingTransport); ok {
			switch kind {
			case CompressGZIP:
				// https://gist.github.com/the42/1956518
				// TODO(njones): https://gist.github.com/erikdubbelboer/7df2b2b9f34f9f839a84
				v.compress = func(fn handlerWithError) handlerWithError {
					return func(w http.ResponseWriter, r *http.Request) error {
						if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
							return fn(w, r)
						}
						w.Header().Set("Content-Encoding", "gzip")

						gz := gzip.NewWriter(w)
						defer gz.Close()

						gzr := compressResponseWriter{Writer: gz, ResponseWriter: w}
						return fn(gzr, r)
					}
				}
			}
		}
	}
}

type quoteWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w quoteWriter) Write(p []byte) (n int, err error) { return w.Writer.Write(p) }

type quoteTransform struct{}

func (q quoteTransform) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	nDst, nSrc = copy(dst, strings.TrimSuffix(strconv.Quote(string(src))[1:], `"`)), len(src)
	if atEOF {
		err = io.EOF
	}
	return
}

func (q quoteTransform) Reset() {}

func jsonp(next handlerWithError) handlerWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		var j string
		if j = r.URL.Query().Get("j"); j == "" {
			return next(w, r)
		}

		tw := transform.NewWriter(w, quoteTransform{})

		w.Header().Set("Content-type", "application/json")
		fmt.Fprintf(w, `___eio[%s]("`, j)

		next(quoteWriter{Writer: tw, ResponseWriter: w}, r)
		tw.Close()

		fmt.Fprint(w, `");`)

		return nil
	}
}

func WithPollingSleep(d time.Duration) Option {
	return func(o OptionWith) {
		if v, ok := o.(*PollingTransport); ok {
			v.sleep = d
		}
	}
}
