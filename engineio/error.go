package engineio

import (
	errs "github.com/njones/socketio/internal/errors"
)

const (
	EOH errs.String = "end of hanshake"

	ErrNoTransport        errs.String = "transport unknown"
	ErrNoSessionID        errs.String = "session id unknown"
	ErrBadHandshakeMethod errs.String = "bad handshake method"
	ErrURIPath            errs.String = "bad URI path"
	ErrTransportRun       errs.String = "bad transport run: %w"
	ErrPayloadEncode      errs.String = "bad payload encode: %w"
)
