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

const EOH erro.String = "end of handshake"
