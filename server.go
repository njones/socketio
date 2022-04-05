package socketio

import (
	"net/http"

	sess "github.com/njones/socketio/session"
)

const (
	// ackIDEventPrefix - is used as an event prefix for acknowledgement ID's
	ackIDEventPrefix = ":\xACk🆔:"
	// socketIDPrefix - is used as a room prefix for sending events to the private socket room
	socketIDPrefix = ":s\x0Cket🆔:"
)

type (
	SocketID = sess.ID

	Namespace = string
	Room      = string
	Event     = string
	Data      = Serializable
)

// Server is the generic interface that's used to designate the socketID as a server
// so that it can be added to a http.Server instance.
type Server = interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// InToEmit is an interface used to limit the next chained method to In, To or Emit
type InToEmit interface {
	In(room Room) InToEmit
	To(room Room) InToEmit
	Emit
}

// BroadcastEmit is an interface used to limit the next chained method to Broadcast or Emit
type BroadcastEmit interface {
	Broadcast() Emit
	Emit
}

// BroadcastEmit is an interface used to limit the next chained method to Emit
type Emit interface {
	Emit(event Event, data ...Data) error
}
