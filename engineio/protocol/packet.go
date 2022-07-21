package protocol

import (
	"encoding/json"
	"io"
)

const (
	OpenPacket PacketType = iota
	ClosePacket
	PingPacket
	PongPacket
	MessagePacket
	UpgradePacket
	NoopPacket
)

const (
	packetTypeLength = int64(len(`0`)) // OpenPacket

	emptyBracketsLength  = len(`{}`)
	commaLength          = len(`,`)
	emptyStringLength    = len(`""`)
	emptySIDLength       = len(`"sid":""`)
	emptyUpgradesLength  = len(`"upgrades":[]`)
	pingTimeoutKeyLength = len(`"pingTimeout":`)
)

type (
	_packetJSONDecoder func(io.Reader) *json.Decoder
	_packetJSONEncoder func(io.Writer) *json.Encoder
)

func (fn _packetJSONDecoder) From(r io.Reader) func(interface{}) error { return fn(r).Decode }
func (fn _packetJSONEncoder) To(w io.Writer) func(interface{}) error {
	return fn(&stripLastNewlineWriter{w}).Encode
}

func newJSONDecoder() _packetJSONDecoder { return json.NewDecoder }
func newJSONEncoder() _packetJSONEncoder { return json.NewEncoder }

type Packet struct {
	T PacketType  `json:"type"`
	D interface{} `json:"data"`

	isOpenPacket bool
}

func (pac Packet) PacketVal() Packet   { return pac }
func (pac *Packet) PacketRef() *Packet { return pac }

type useLen interface{ Len() int }

func (pac Packet) Len() int {
	var n = 1 // the length of the type
	switch d := pac.D.(type) {
	case nil:
		return n
	case string:
		return n + len([]rune(d))
	case useLen:
		return n + d.Len()
	}
	return 0
}

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
