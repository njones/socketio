package transport

import (
	"io"
	"strings"

	eiop "github.com/njones/socketio/engineio/protocol"
	eios "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
	siop "github.com/njones/socketio/protocol"
	sios "github.com/njones/socketio/session"
)

type (
	// The EngineIO session ID
	SessionID = eios.ID

	// The SocketID session ID
	SocketID = sios.ID

	// The functional option that can be used with Packets
	Option = siop.Option

	Namespace = string
	Room      = string

	Data interface{} // The Data packet type

	// Socket is a generic socket that is passed to the emit function during execution
	Socket struct {
		Type      byte
		Namespace string
		AckID     uint64
		Data      Data
	}
)

// buffer holds EngineIO packets until the buffer is stopped. This is used
// when waiting to send a connection packet back first even though it may
// not be processed first.
type buffer struct {
	active  bool
	packets []eiop.Packet
}

// StartBuffer starts buffering EngineIO packets
func (buf *buffer) StartBuffer() func() {
	buf.active = true
	return buf.StopBuffer
}

// StopBuffer stops buffering EngineIO packets
func (buf *buffer) StopBuffer() { buf.active = false }

// Transport facilitates transferring between the SocketIO transport which
// is in-memory or something like redis to the EngineIO transport which is
// HTTP long polling, Websockets or Server-Side events
type Transport struct {
	id SocketID

	*buffer

	receive        chan Socket
	newPacket      siop.NewPacket
	eioTransport   eiot.Transporter
	isDisconnected bool
}

func NewTransport(id SocketID, eioTransport eiot.Transporter, fn siop.NewPacket) *Transport {
	return &Transport{
		buffer:       &buffer{},
		id:           id,
		receive:      make(chan Socket, 1000),
		newPacket:    fn,
		eioTransport: eioTransport,
	}
}

func (t *Transport) Disconnect() {
	t.isDisconnected = true
}

func (t *Transport) IsDisconnected() bool {
	return t.isDisconnected
}

func (t *Transport) SendBuffer() {
	for _, packet := range t.buffer.packets {
		t.eioTransport.Send(packet)
		t.sendBinary(packet)
	}
	t.buffer.packets = t.buffer.packets[:0] // clear the buffer
}

func (t *Transport) Send(data Data, opts ...Option) {
	sioPacket := t.newPacket().WithData(data).WithOption(opts...)
	eioPacket := eiop.Packet{T: eiop.MessagePacket, D: sioPacket}
	if t.buffer.active {
		t.buffer.packets = append(t.buffer.packets, eioPacket)
		return
	}

	t.eioTransport.Send(eioPacket)
	t.sendBinary(eioPacket)
}

func (t *Transport) sendBinary(packet eiop.Packet) {
	if pac, ok := packet.D.(siop.Packet).(interface{ GetData() interface{} }); ok {
		objs, _ := pac.GetData().([]interface{})
		for _, v := range objs {
			if r, ok := v.(io.Reader); ok {
				eioBinaryPacket := eiop.Packet{T: eiop.BinaryPacket, D: r}
				t.eioTransport.Send(eioBinaryPacket)
			}
		}
	}
}

func (t *Transport) Receive() <-chan Socket {
	go func() {
		for eioPacket := range t.eioTransport.Receive() {
			switch data := eioPacket.D.(type) {
			case string:
				pac := t.newPacket().(packet)
				if _, err := pac.(io.ReaderFrom).ReadFrom(strings.NewReader(data)); err != nil {
					t.eioTransport.Send(eiop.Packet{T: eiop.NoopPacket, D: err})
				}

				switch pac.GetType() {
				case siop.BinaryEventPacket.Byte(), siop.BinaryAckPacket.Byte():
					if in, ok := pac.(interface{ ReadBinary() func(io.Reader) error }); ok {

						t.receive <- packetToSocket(pac)
						var cntPlaceholders int
					EIOPacketData:
						for eioPacket = range t.eioTransport.Receive() {
							bin := in.ReadBinary()
							if r, ok := eioPacket.D.(io.Reader); ok && bin != nil {
								bin(r)
							}

							cntPlaceholders++
							if cntPlaceholders >= len(pac.GetData().([]interface{}))-1 || // TODO(njones): base this off of the binary index...
								eioPacket.T != eiop.BinaryPacket {
								break EIOPacketData
							}
						}
						continue
					}
				}

				t.receive <- packetToSocket(pac)
			}

			switch eioPacket.T {
			case eiop.NoopPacket:
				if done, ok := eioPacket.D.(interface{ SocketCloseChannel() error }); ok {

					if err := done.SocketCloseChannel(); err != nil {
						sioPacket := t.newPacket().
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
