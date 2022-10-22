package engineio

import (
	"net/http"
	"strconv"

	eios "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
)

type ctxKey string

const ctxSessionID ctxKey = "sessionID"
const ctxTransportName ctxKey = "transportName"
const ctxEIOVersion ctxKey = "eioVersion"

type (
	SessionID     = eios.ID
	TransportName = eiot.Name

	EIOVersionStr string
	EIOVersionInt int
)

func (v EIOVersionStr) Int() EIOVersionInt { i, _ := strconv.Atoi(string(v)); return EIOVersionInt(i) }

type server interface {
	Server
	serveTransport(http.ResponseWriter, *http.Request) (eiot.Transporter, error)
}

type Server = interface {
	OptionWith
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type EIOServer interface {
	Server
	ServeTransport(http.ResponseWriter, *http.Request) (eiot.Transporter, error)
}

func NewServer(opts ...Option) Server { return registry.latest(opts...) }
