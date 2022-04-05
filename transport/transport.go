package transport

import (
	"strings"

	eiop "github.com/njones/socketio/engineio/protocol"
	eess "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
	siop "github.com/njones/socketio/protocol"
	sess "github.com/njones/socketio/session"
)

type (
	SessionID = eess.ID
	SocketID  = sess.ID
	Option    = siop.Option

	Namespace = string
	Room      = string

	Data interface{}

	Socket struct {
		Type      byte
		Namespace string
		AckID     uint64
		Data      Data
	}
)

type Transport struct {
	id SocketID

	receive      chan Socket
	packetMaker  siop.NewPacket
	eioTransport eiot.Transporter
}

func NewTransport(id SocketID, eioTransport eiot.Transporter, packetMaker siop.NewPacket) *Transport {
	return &Transport{
		id:           id,
		receive:      make(chan Socket, 1000),
		packetMaker:  packetMaker,
		eioTransport: eioTransport,
	}
}

func (t *Transport) Send(data Data, opts ...Option) {
	sioPacket := t.packetMaker().WithData(data).WithOption(opts...)
	t.eioTransport.Send(eiop.Packet{T: eiop.MessagePacket, D: sioPacket})
}

func (t *Transport) Receive() <-chan Socket {
	go func() {
		for eioPacket := range t.eioTransport.Receive() {

			if eioPacket.T == eiop.NoopPacket {
				if done, ok := eioPacket.D.(interface{ SocketCloseChannel() error }); ok {

					if err := done.SocketCloseChannel(); err != nil {
						sioPacket := t.packetMaker().
							WithType(siop.ErrorPacket.Byte()).
							WithData(err)
						t.receive <- packetToSocket(sioPacket.(packet))
					}

					close(t.receive)
					return
				}
			}

			switch data := eioPacket.D.(type) {
			case string:
				sioPacket := t.packetMaker().(siop.PacketReadWrite)
				if _, err := sioPacket.ReadFrom(strings.NewReader(data)); err != nil {
					t.eioTransport.Send(eiop.Packet{T: eiop.NoopPacket, D: err})
				}

				t.receive <- packetToSocket(sioPacket.(packet))
			}

		}
	}()
	return t.receive
}
