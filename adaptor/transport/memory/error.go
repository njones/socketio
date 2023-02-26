package memory

import (
	erro "github.com/njones/socketio/internal/errors"
)

// All of the possible errors the map transport can return
const (
	ErrSocketIDTransportNotFound erro.StringF = "socket id %q not found in the in-memory map"
	ErrNilTransporter            erro.String  = "expected a type of Transporter, found <nil>"
)
