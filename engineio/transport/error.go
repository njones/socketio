package transport

import erro "github.com/njones/socketio/internal/errors"

const (
	ErrDecodeFailed        erro.StringF = "failed to decode the %q transport:: %w"
	ErrEncodeFailed        erro.StringF = "failed to encode the %q transport:: %w"
	ErrUnimplementedMethod erro.StringF = "unimplemented %s method"
	ErrCloseSocket         erro.String  = "socket: closed"
	ErrTimeoutSocket       erro.String  = "socket: timeout"
)
