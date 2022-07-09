package tmap

import (
	errs "github.com/njones/socketio/internal/errors"
)

// All of the possible errors the map transport can return
const (
	ErrInvalidSocketTransport errs.String = "invalid %s transport for socket"
	ErrNilTransporter         errs.String = "cannot add a nil transporter"
)
