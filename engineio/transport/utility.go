package transport

import "net/http"

type socketClose struct{ error }

func (sc socketClose) SocketCloseChannel() error { return sc.error }

func jsonpFrom(r *http.Request) *string {
	j := r.URL.Query().Get("j")
	if j != "" {
		return &j
	}
	return nil
}
