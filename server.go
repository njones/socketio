package socketio

import (
	"encoding/hex"
	"math/rand"
	"net/http"
	"time"

	eios "github.com/njones/socketio/engineio/session"
	seri "github.com/njones/socketio/serialize"
	sios "github.com/njones/socketio/session"
	siot "github.com/njones/socketio/transport"
)

const (
	// ackIDEventPrefix - is used as an event prefix for acknowledgement ID's
	ackIDEventPrefix = ":\xACk🆔:"
	// socketIDPrefix - is used as a room prefix for sending events to the private socket room
	socketIDPrefix = ":s\x0Cket🆔:"
)

const (
	OnDisconnectEvent = "disconnect"
)

type (
	// SocketID is am alias of a session id, so that we don't need to
	// reference sessions through the package
	SessionID = eios.ID
	SocketID  = sios.ID

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

// eventCallback is the callback that is used when an event is called
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

type rawTransport interface {
	Transport(siot.SocketID) *siot.Transport
}

var _socketIDQuickPrefix = func(now time.Time) func() string {
	return func() string {
		src := rand.NewSource(now.UnixNano())
		rnd := rand.New(src)

		cards := [][]rune{
			{127137, 127150}, // spades
			{127153, 127166}, // hearts
			{127169, 127182}, // diamonds
			{127185, 127198}, // clubs
		}

		prefix := make([]rune, 5)
		for i := range prefix {
			suit := rnd.Intn(4)
			card := int32(rnd.Intn(int(cards[suit][1]-cards[suit][0]-1))) + cards[suit][0]
			prefix[i] = card
		}

		enc := hex.EncodeToString([]byte(string(prefix)))
		return enc + "::"
	}
}

var socketIDQuickPrefix = _socketIDQuickPrefix(time.Now())
