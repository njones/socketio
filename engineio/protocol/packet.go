package protocol

import "io"

const (
	OpenPacket PacketType = iota
	ClosePacket
	PingPacket
	PongPacket
	MessagePacket
	UpgradePacket
	NoopPacket

	BinaryPacket PacketType = 255
)

type Packet struct {
	T PacketType  `json:"type"`
	D interface{} `json:"data"`
}

func (pac Packet) PacketVal() Packet   { return pac }
func (pac *Packet) PacketRef() *Packet { return pac }

type PacketType byte

func (pt PacketType) Bytes() []byte {
	if pt == BinaryPacket {
		return []byte{'b'}
	}
	return []byte{byte(pt) + '0'}
}

func (pt *PacketType) Read(p []byte) (n int, err error) {
	for ; len(p) > 0 && n < 1; n++ {
		switch *pt {
		case BinaryPacket:
			p[n] = 'b'
		default:
			p[n] = byte(*pt) + '0'
		}
	}
	return n, nil
}

func (pt *PacketType) Write(p []byte) (n int, err error) {
	for ; len(p) > 0 && n < 1; n++ {
		switch p[n] {
		case 'b':
			*pt = PacketType(255)
		default:
			*pt = PacketType(byte(p[n] & 0x0F))
		}
	}
	return n, nil
}

func (pt PacketType) String() string {
	switch pt {
	case OpenPacket:
		return "open"
	case ClosePacket:
		return "close"
	case PingPacket:
		return "ping"
	case PongPacket:
		return "pong"
	case MessagePacket:
		return "message"
	case UpgradePacket:
		return "upgrade"
	case NoopPacket:
		return "noop"
	case BinaryPacket:
		return "binary message"
	}
	return "unknown packet type"
}

type (
	PacketEncoder interface{ To(io.Writer) PacketWriter }
	PacketDecoder interface{ From(io.Reader) PacketReader }

	PacketWriter interface{ WritePacket(PacketVal) error }
	PacketReader interface{ ReadPacket(PacketRef) error }

	PacketVal interface{ PacketVal() Packet }
	PacketRef interface{ PacketRef() *Packet }
)
