package socketio

import (
	"net/http"

	sess "github.com/njones/socketio/session"
)

const (
	ackIDEventPrefix = ":\xACk🆔:"
	socketIDPrefix   = ":s\x0Cket🆔:"
)

type (
	SocketID = sess.ID

	Namespace = string
	Room      = string
	Event     = string
	Data      = Serializable
)

type Server interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type InToEmit interface {
	In(room Room) InToEmit
	To(room Room) InToEmit
	Emit
}

type BroadcastEmit interface {
	Broadcast() Emit
	Emit
}

type Emit interface {
	Emit(event Event, data ...Data) error
}
