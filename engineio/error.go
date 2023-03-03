package engineio

import (
	"strings"

	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrUnknownTransport         = httpErrStr(erro.HTTPStatusError400 + "unknown transport")
	ErrUnknownSessionID         = httpErrStr(erro.HTTPStatusError400 + "unknown session id")
	ErrUnknownEIOVersion        = httpErrStr(erro.HTTPStatusError400 + "unknown engineio version")
	ErrInvalidRequestHTTPMethod = httpErrStr(erro.HTTPStatusError400 + "invalid request, an unimplemented HTTP method")
	ErrInvalidURIPath           = httpErrStr(erro.HTTPStatusError400 + "invalid URI path, the prefix is not found")
	ErrTransportUpgradeFailed   = httpErrStr(erro.HTTPStatusError400 + "failed to upgrade transport")

	EOH erro.State = "End Of Handshake"
	IOR erro.State = "Is OPTION Request"
)

type httpErrStr string

func (e httpErrStr) Error() string { return string(e[erro.HTTPStatusErrorLen:]) }

func (e httpErrStr) Is(target error) bool {
	if prefix, ok := target.(erro.StatusError); ok {
		return strings.HasPrefix(string(e), string(prefix))
	}
	return false
}
