package engineio

import (
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

const EOH erro.String = "end of handshake"
const IOR erro.String = "is OPTION request"
