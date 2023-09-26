package transport

import eiot "github.com/njones/socketio/engineio/transport"

type packet interface {
	GetType() byte
	GetNamespace() string
	GetAckID() uint64
	GetData() interface{}
}

type Transporter interface {
	AddSetter
	JoinLeaver
	SendReceiver

	AckID() uint64

	GetSocketID(eiot.SessionID) *SocketID
	Disconnect(SocketID)
	IsDisconnected(SocketID) bool
}

type Sender interface {
	Send(SocketID, Data, ...Option) error
}

type SendReceiver interface {
	Sender
	Receive(socketID SocketID) <-chan Socket
}

type JoinLeaver interface {
	Join(Namespace, SocketID, Room) error
	Leave(Namespace, SocketID, Room) error
}

type AddSetter interface {
	Add(eiot.Transporter) (SocketID, error)
	Set(SocketID, eiot.Transporter) error
}

type Emitter interface {
	Sender

	Sockets(ns Namespace) SocketArray
	Rooms(ns Namespace, id SocketID) RoomArray
}
