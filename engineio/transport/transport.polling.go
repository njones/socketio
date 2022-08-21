package transport

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	"golang.org/x/text/transform"
)

type handlerWithError func(http.ResponseWriter, *http.Request) error

type PollingTransport struct {
	*Transport

	interval time.Duration
	sleep    time.Duration

	compress func(handlerWithError) handlerWithError
}

func NewPollingTransport(chanBuf int, ival time.Duration) func(SessionID, Codec) Transporter {
	return func(id SessionID, codec Codec) Transporter {
		t := &PollingTransport{
			Transport: &Transport{
				id:      id,
				name:    "polling",
				codec:   codec,
				send:    make(chan eiop.Packet, chanBuf),
				receive: make(chan eiop.Packet, chanBuf),
			},
			compress: func(fn handlerWithError) handlerWithError {
				return func(w http.ResponseWriter, r *http.Request) error {
					return fn(w, r)
				}
			},
			interval: ival,
			sleep:    25 * time.Millisecond,
		}

		return t
	}
}

func (t *PollingTransport) Run(_w http.ResponseWriter, r *http.Request, opts ...Option) (err error) {
	for _, opt := range opts {
		opt(t)
	}

	w := &writer{ResponseWriter: _w}
	defer func() {
		if errors.Is(err, http.ErrBodyReadAfterClose) && !w.DataWritten() {
			err = nil
		}

		if err != nil {
			t.onErr <- err
		}
	}()

	ctx, cancel := context.WithCancel(r.Context())
	t.Transport.shutdown = func() { cancel() }

	switch r.Method {
	case http.MethodGet:
		return t.compress(jsonp(t.poll))(w, r.WithContext(ctx))
	case http.MethodPost:
		// decompression will happen automatically
		return t.emit(w, r.WithContext(ctx))
	}
	return nil

}

// longPoll allows a connection for a specified amout of time... then releases a payload
func (t *PollingTransport) poll(w http.ResponseWriter, r *http.Request) error {
	var ctx = r.Context()
	var interval = time.After(t.interval)
	var packets eiop.Payload

Write:
	for {
		select {
		case packet := <-t.receive:
			packets = append(packets, packet)
		case <-ctx.Done():
			break Write
		case <-interval:
			break Write
		default:
			time.Sleep(t.sleep) // let other things come in if things are coming quick...
			if len(packets) > 0 && len(t.receive) == 0 {
				break Write
			}
		}
	}

	if len(packets) > 0 {
		if err := t.codec.PayloadEncoder.To(w).WritePayload(packets); err != nil {
			t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{err}}
			return ErrTransportEncode.F("polling", err)
		}
	}

	t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{}} // shutdown the HTTP connection
	return nil
}

// gather pulls in all of the posts
func (t *PollingTransport) emit(w http.ResponseWriter, r *http.Request) error {
	var payload eiop.Payload
	if err := t.codec.PayloadDecoder.From(r.Body).ReadPayload(&payload); err != nil {
		t.send <- eiop.Packet{T: eiop.NoopPacket, D: socketClose{err}}
		return ErrTransportDecode.F("polling", err)
	}

	for _, packet := range payload {
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
	return func(t Transporter) {
		switch v := t.(type) {
		case *PollingTransport:
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
		default:
			// show log of no compression used...
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
