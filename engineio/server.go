package engineio

import (
	"net/http"
	"strconv"

	sess "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
)

type (
	SessionID = sess.ID

	EIOVersionStr string
	EIOVersionInt int
)

func (v EIOVersionStr) Int() EIOVersionInt { i, _ := strconv.Atoi(string(v)); return EIOVersionInt(i) }

type server interface {
	Server
	serveTransport(http.ResponseWriter, *http.Request) (eiot.Transporter, error)
}

type Server = interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type EIOServer interface {
	Server
	ServeTransport(http.ResponseWriter, *http.Request) (eiot.Transporter, error)
}

func NewServer(opts ...Option) Server { return registry.latest(opts...) }
