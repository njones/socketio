package transport

import erro "github.com/njones/socketio/internal/errors"

const (
	ErrTransportDecode   erro.String = "[%s] transport decode: %w"
	ErrTransportEncode   erro.String = "[%s] transport encode: %w"
	ErrUnsupportedMethod erro.String = "%s not supported"
)
