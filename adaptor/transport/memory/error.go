package memory

import (
	erro "github.com/njones/socketio/internal/errors"
)

// All of the possible errors the map transport can return
const (
	ErrInvalidSocketTransport erro.String = "invalid %s transport for socket"
	ErrNilTransporter         erro.String = "cannot add a nil transporter"
)
