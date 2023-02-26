package engineio

import (
	"fmt"

	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrUnknownTransport  erro.String = "unknown transport"
	ErrUnknownSessionID  erro.String = "unknown session id"
	ErrUnknownEIOVersion erro.String = "unknown engineio version"
	ErrRequestHTTPMethod erro.String = "invalid request, an unimplemented HTTP method"
	ErrURIPath           erro.String = "invalid URI path, the prefix is not found"

	EOH erro.String = "End Of Handshake"
	IOR erro.String = "Is OPTION Request"
)

const HTTPStatusError400 httpErrorStatus = 400

var ErrBadUpgrade = httpError{400, "failed to upgrade transport"}

type httpErrorStatus int

func (e httpErrorStatus) Error() string { return fmt.Sprintf("HTTP status: %d", e) }

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
