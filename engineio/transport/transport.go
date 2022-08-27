package transport

import (
	"net/http"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	eios "github.com/njones/socketio/engineio/session"
)

type SessionID = eios.ID

type Name string

func (name Name) String() string { return string(name) }

type Codec struct {
	eiop.PacketEncoder
	eiop.PacketDecoder
	eiop.PayloadEncoder
	eiop.PayloadDecoder
}

type Transporter interface {
	ID() SessionID
	Name() Name
	Send(eiop.Packet)
	Receive() <-chan eiop.Packet

	Run(http.ResponseWriter, *http.Request, ...Option) error

	Shutdown()
}

type Transport struct {
	id    SessionID
	name  Name
	codec Codec

	pingTimeout  time.Duration
	pingInterval time.Duration

	send, receive chan eiop.Packet
	onErr         chan error

	shutdown func()
}

func (t *Transport) ID() SessionID               { return t.id }
func (t *Transport) Name() Name                  { return t.name }
func (t *Transport) Send(packet eiop.Packet)     { t.receive <- packet }
func (t *Transport) Receive() <-chan eiop.Packet { return t.send }
func (t *Transport) Transport() *Transport       { return t }
func (t *Transport) Shutdown() {
	if t.shutdown != nil {
		t.shutdown()
	}
}
