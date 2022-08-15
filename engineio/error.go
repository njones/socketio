package engineio

import (
	errs "github.com/njones/socketio/internal/errors"
)

const (
	ErrNoTransport        errs.String = "transport unknown"
	ErrNoSessionID        errs.String = "session id unknown"
	ErrBadHandshakeMethod errs.String = "bad handshake method"
	ErrURIPath            errs.String = "bad URI path"
	ErrTransportRun       errs.String = "bad transport run: %w"
	ErrPayloadEncode      errs.String = "bad payload encode: %w"
)

type EndOfHandshake struct{ SessionID string }

func (e EndOfHandshake) Is(err error) bool {
	_, ok := err.(EndOfHandshake)
	return ok
}

func (e EndOfHandshake) Error() string { return "end of handshake" }
