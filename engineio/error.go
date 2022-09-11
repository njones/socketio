package engineio

import (
	"fmt"

	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrNoTransport        erro.String = "transport unknown"
	ErrNoSessionID        erro.String = "session id unknown"
	ErrNoEIOVersion       erro.String = "eio version unknown"
	ErrBadHandshakeMethod erro.String = "bad handshake method"
	ErrBadRequestMethod   erro.String = "bad http request method"
	ErrURIPath            erro.String = "bad URI path"
	ErrTransportRun       erro.String = "bad transport run: %w"
	ErrPayloadEncode      erro.String = "bad payload encode: %w"
)

const HTTPStatusError400 httpErrorStatus = 400

var ErrBadUpgrade = httpError{400, "transport upgrade error"}

const EOH erro.String = "end of handshake"
const IOR erro.String = "is OPTION request"

type httpErrorStatus int

func (e httpErrorStatus) Error() string { return fmt.Sprintf("http error: %d", e) }

type httpError struct {
	status int
	erro.String
}

func (e httpError) Error() string { return string(e.String) }

func (e httpError) Is(target error) bool {
	if te, ok := target.(httpErrorStatus); ok {
		return int(te) == e.status
	}
	return false
}
