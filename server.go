package socketio

import (
	"net/http"

	seri "github.com/njones/socketio/serialize"
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
	Data      = seri.Serializable
)

// Server is the generic interface that's used to designate the socketID as a server
// so that it can be added to a http.Server instance.
type Server = interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type eventCallback interface {
	Callback(...interface{}) error
}

// inToEmit is an interface used to limit the next chained method to In, To or Emit
type inToEmit interface {
	In(room Room) inToEmit
	To(room Room) inToEmit
	emit
}

// broadcastEmit is an interface used to limit the next chained method to Broadcast or Emit
type broadcastEmit interface {
	Broadcast() emit
	emit
}

// broadcastEmit is an interface used to limit the next chained method to Emit
type emit interface {
	Emit(event Event, data ...Data) error
}
