package transport

import (
	"io"
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

type buffer struct {
	active  bool
	packets []eiop.Packet
}

func (buf *buffer) StartBuffer() { buf.active = true }
func (buf *buffer) StopBuffer()  { buf.active = false }

type Transport struct {
	id SocketID

	*buffer

	receive      chan Socket
	packetMaker  siop.NewPacket
	eioTransport eiot.Transporter
}

func NewTransport(id SocketID, eioTransport eiot.Transporter, packetMaker siop.NewPacket) *Transport {
	return &Transport{
		buffer:       &buffer{},
		id:           id,
		receive:      make(chan Socket, 1000),
		packetMaker:  packetMaker,
		eioTransport: eioTransport,
	}
}

func (t *Transport) SendBuffer() {
	for _, packet := range t.buffer.packets {
		t.eioTransport.Send(packet)
	}
}

func (t *Transport) Send(data Data, opts ...Option) {
	sioPacket := t.packetMaker().WithData(data).WithOption(opts...)
	eioPacket := eiop.Packet{T: eiop.MessagePacket, D: sioPacket}
	if t.buffer.active {
		t.buffer.packets = append(t.buffer.packets, eioPacket)
		return
	}
	t.eioTransport.Send(eioPacket)
}

func (t *Transport) Receive() <-chan Socket {
	go func() {
		for eioPacket := range t.eioTransport.Receive() {

			switch data := eioPacket.D.(type) {
			case string:
				sioPacket := t.packetMaker().(io.ReaderFrom)
				if _, err := sioPacket.ReadFrom(strings.NewReader(data)); err != nil {
					t.eioTransport.Send(eiop.Packet{T: eiop.NoopPacket, D: err})
				}
				if pac, ok := sioPacket.(interface{ GetType() byte }); ok {
					switch pac.GetType() {
					case siop.BinaryEventPacket.Byte(), siop.BinaryAckPacket.Byte():
						if in, ok := sioPacket.(interface{ ReadBinary() func(io.Reader) error }); ok {
						EIOPacketData:
							for eioPacket = range t.eioTransport.Receive() {
								if eioPacket.T != eiop.BinaryPacket {
									break EIOPacketData
								}

								bin := in.ReadBinary()
								if r, ok := eioPacket.D.(io.Reader); ok && bin != nil {
									bin(r)
								}
							}

						}
					}
				}

				t.receive <- packetToSocket(sioPacket.(packet))
			}

			switch eioPacket.T {
			case eiop.NoopPacket:
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

		}
	}()
	return t.receive
}
