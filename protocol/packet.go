package protocol

// Packet is the interface for objects that can be passed around
// as socket.io packets. It provides a fluent interface for adding
// data common data to the underling type. The WithOption method
// can be used to add data to underling types that have additional
// data than that standard, type, namespace, ackID and data.
type Packet interface {
	WithOption(...Option) Packet

	WithType(byte) Packet
	WithNamespace(string) Packet
	WithAckID(uint64) Packet
	WithData(interface{}) Packet
}

// NewPacket provides an external type for a Packet factory function.
// This is mainly used when creating new socket.io protocol transports.
type NewPacket func() Packet

// packet represents a basic socket.io packet of data (v0 through v5)
type packet struct {
	Type      packetType  `json:"type"`
	Namespace packetNS    `json:"nsp"`
	AckID     packetAckID `json:"id"`
	Data      packetData  `json:"data"`

	ket func() Packet `json:"-"` // is a function that will return self packet (object) as a Packet (interface)
}

func (pac packet) Len() (n int) {
	n += pac.Type.Len()
	m := pac.Namespace.Len()
	n += m
	n += pac.AckID.Len()
	if x, ok := pac.Data.(interface{ Len() int }); ok {
		n += x.Len()
	}
	if m > 0 && n > m+1 {
		n += 1 // for the extra namespace comma
	}
	return n
}

// -------------------------------------------------
// -- provide the Packet interface methods below ---
// -------------------------------------------------

// WithOption applies all of the Option functions to a packet object, then returns the Packet interface. This
// method satisfies the interface for a Packet, so it allows packet(s) to be Packet(s).
func (pac *packet) WithOption(opts ...Option) Packet {
	for _, opt := range opts {
		opt(pac)
	}
	return pac.ket()
}

// WithType sets the packet Type to the x byte (the underlining type for packetType) type. This is so that
// external packages don't need to know about protocol.PacketType types, but just the basic underlining type,
// we will convert it to the correct type.
func (pac *packet) WithType(x byte) Packet { pac.Type = packetType(x); return pac.ket() }

// WithNamespace sets the packet Namespace to the x string (the underlining type for packetNS) type. This is so that
// external packages don't need to know about protocol.PacketType types, but just the basic underlining type,
// we will convert it to the correct type.
func (pac *packet) WithNamespace(x string) Packet { pac.Namespace = packetNS(x); return pac.ket() }

// WithAckID sets the packet AckID to the x string (the underlining type for packetAckID) type. This is so that
// external packages don't need to know about protocol.PacketAckID types, but just the basic underlining type,
// we will convert it to the correct type.
func (pac *packet) WithAckID(x uint64) Packet { pac.AckID = packetAckID(x); return pac.ket() }

// WithData sets the packet Namespace to the x interface{} (the underlining type for packetData) type. This is so that
// external packages don't need to know about protocol.PacketDataString, protocol.PacketDataArray or protocol.PacketDataObject
// types, but just the basic underlining type, we will convert it to the correct type.
func (pac *packet) WithData(x interface{}) Packet { pac.Data = withPacketData(x); return pac.ket() }

// -------------------------------------------------

// GetType returns the underlining byte type for a socket.io packet Type
func (pac *packet) GetType() byte { return byte(pac.Type) }

// GetNamespace returns the underlining string type for a socket.io packet Namespace
func (pac *packet) GetNamespace() string { return string(pac.Namespace) }

// GetAckID returns the underlining string type for a socket.io packet AckID
func (pac *packet) GetAckID() uint64 { return uint64(pac.AckID) }

// GetData returns the underlining data string/array type for a socket.io packet Data
func (pac *packet) GetData() interface{} {
	switch val := pac.Data.(type) {
	case *packetDataString:
		return val.x // unwrap and return the string
	case *packetDataArray:
		return val.x // unwrap and return the array
	case *packetDataObject:
		return val.x // unwrap and return the object map
	}
	return pac.Data // returns the encapsulated, possibly wrapped data
}
