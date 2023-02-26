package transport

import (
	"bufio"
	"net"
	"net/http"
)

type writer struct {
	header, body bool
	http.ResponseWriter
}

func (wr *writer) Write(p []byte) (int, error) {
	wr.body = true
	return wr.ResponseWriter.Write(p)
}

func (wr *writer) WriteHeader(s int) {
	wr.header = true
	wr.ResponseWriter.WriteHeader(s)
}

func (w *writer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, ErrUnimplementedMethod.F("http.Hijacker()")
	}
	return h.Hijack()
}

func (wr *writer) DataWritten() bool {
	return (wr.body != wr.header) || wr.body
}
