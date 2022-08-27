package engineio

import (
	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrNoTransport        erro.String = "transport unknown"
	ErrNoSessionID        erro.String = "session id unknown"
	ErrBadHandshakeMethod erro.String = "bad handshake method"
	ErrURIPath            erro.String = "bad URI path"
	ErrTransportRun       erro.String = "bad transport run: %w"
	ErrPayloadEncode      erro.String = "bad payload encode: %w"
)

type EndOfHandshake struct{ SessionID string }

func (e EndOfHandshake) Is(err error) bool {
	_, ok := err.(EndOfHandshake)
	return ok
}

func (e EndOfHandshake) Error() string { return "end of handshake" }
