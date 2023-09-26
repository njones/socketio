package transport

import (
	"net/http"

	eiop "github.com/njones/socketio/engineio/protocol"
	eios "github.com/njones/socketio/engineio/session"
)

type SessionID = eios.ID

type Name string

func (name Name) String() string { return string(name) }

type WaitGroup interface {
	Add(int)
	Done()
	Wait()
}

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
	SendTimeout()
	ReceiveTimeout() <-chan SessionID

	Run(http.ResponseWriter, *http.Request, ...Option) error

	Shutdown()
}

type StartWriteBuffer func() bool

func (StartWriteBuffer) Len() int { return 0 }

type Transport struct {
	id    SessionID
	name  Name
	codec Codec

	sendPing bool

	send, receive chan eiop.Packet
	expireId      chan SessionID

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

func (t *Transport) SendTimeout()                     { t.expireId <- t.id }
func (t *Transport) ReceiveTimeout() <-chan SessionID { return t.expireId }
