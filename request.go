package socketio

import (
	"context"
	"net/http"
	"net/url"
)

// Request is a wrapped HTTP request object so that we expose only the things that are necessary,
type Request struct {
	r *http.Request

	Method     string
	URL        *url.URL
	Header     http.Header
	Host       string
	RemoteAddr string
	RequestURI string
}

func (req *Request) Cookie(name string) (*http.Cookie, error) { return req.r.Cookie(name) }
func (req *Request) Cookies() []*http.Cookie                  { return req.r.Cookies() }
func (req *Request) Context() context.Context                 { return req.r.Context() }
func (req *Request) Referer() string                          { return req.r.Referer() }
func (req *Request) UserAgent() string                        { return req.r.UserAgent() }
func (req *Request) WithContext(ctx context.Context) *Request {
	req.r = req.r.WithContext(ctx)
	return req
}

func sioRequest(r *http.Request) *Request {
	req := &Request{
		r:          r,
		Method:     r.Method,
		URL:        r.URL,
		Header:     r.Header,
		Host:       r.Host,
		RemoteAddr: r.RemoteAddr,
		RequestURI: r.RequestURI,
	}
	return req
}
