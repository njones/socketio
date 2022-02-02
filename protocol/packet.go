package protocol

import "io"

type Packet interface {
	WithOption(...Option) Packet

	WithType(byte) Packet
	WithNamespace(string) Packet
	WithAckID(uint64) Packet
	WithData(interface{}) Packet
}

type PacketReadWrite interface {
	io.ReaderFrom
	io.WriterTo
}

type NewPacket func() Packet

type packet struct {
	Type      packetType  `json:"type"`
	Namespace packetNS    `json:"nsp"`
	AckID     packetAckID `json:"id"`
	Data      packetData  `json:"data"`

	ket func() Packet `json:"-"`
}

// provides the interface for defining the values of a basic packet

func (pac *packet) WithOption(opts ...Option) Packet {
	for _, opt := range opts {
		opt(pac)
	}
	return pac.ket()
}

func (pac *packet) WithType(x byte) Packet        { pac.Type = packetType(x); return pac.ket() }
func (pac *packet) WithNamespace(x string) Packet { pac.Namespace = packetNS(x); return pac.ket() }
func (pac *packet) WithAckID(x uint64) Packet     { pac.AckID = packetAckID(x); return pac.ket() }
func (pac *packet) WithData(x interface{}) Packet { pac.Data = withPacketData(x); return pac.ket() }

func (pac *packet) GetType() byte        { return byte(pac.Type) }
func (pac *packet) GetNamespace() string { return string(pac.Namespace) }
func (pac *packet) GetAckID() uint64     { return uint64(pac.AckID) }
func (pac *packet) GetData() interface{} {
	switch val := pac.Data.(type) {
	case *packetDataString:
		return val.x
	case *packetDataArray:
		return val.x
	}
	return pac.Data
}
