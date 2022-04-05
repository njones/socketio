package transport

import errs "github.com/njones/socketio/internal/errors"

const (
	ErrTransportDecode   errs.String = "[%s] transport decode: %w"
	ErrTransportEncode   errs.String = "[%s] transport encode: %w"
	ErrUnsupportedMethod errs.String = "%s not supported"
)
