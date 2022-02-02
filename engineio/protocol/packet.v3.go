package protocol

import (
	"encoding/json"
	"fmt"
	"io"
)

// PacketV3 is defined: https://github.com/socketio/engine.io-protocol/tree/v3
type PacketV3 struct {
	Packet

	IsBinary bool
}

type PacketDecoderV3 struct{ *PacketDecoderV2 }

var NewPacketDecoderV3 _packetDecoderV3 = func(r io.Reader) *PacketDecoderV3 {
	return &PacketDecoderV3{PacketDecoderV2: &PacketDecoderV2{r: r}}
}

func (dec *PacketDecoderV3) Decode(packet *PacketV3) error {
	if packet == nil {
		packet = &PacketV3{}
	}
	if packet.T == 0 {
		if _, err := io.CopyN(&packet.T, dec.r, 1); err != nil {
			return ErrPacketDecode.F("v3", err)
		}
	}

	switch packet.T {
	case OpenPacket:
		var data HandshakeV3
		if err := json.NewDecoder(dec.r).Decode(&data); err != nil {
			return ErrHandshakeDecode.F("v3", err)
		}
		packet.D = data
	case BinaryPacket, MessagePacket:
		if packet.IsBinary {
			data, err := io.ReadAll(dec.r)
			if err != nil {
				return ErrPacketDecode.F("v3", err)
			}
			packet.D = data
			return nil
		}
		fallthrough
	case PingPacket, PongPacket, ClosePacket, UpgradePacket, NoopPacket:
		var v2 = PacketV2{Packet: packet.Packet}
		if err := dec.PacketDecoderV2.Decode(&v2); err != nil {
			return ErrPacketDecode.F("v3", err)
		}
		packet.D = v2.D // the passed in packet is not a reference so we need add it here
	}

	return nil
}

type PacketEncoderV3 struct{ *PacketEncoderV2 }

var NewPacketEncoderV3 _packetEncoderV3 = func(w io.Writer) *PacketEncoderV3 {
	return &PacketEncoderV3{PacketEncoderV2: &PacketEncoderV2{w: w}}
}

func (enc *PacketEncoderV3) Encode(packet PacketV3) (err error) {
	switch packet.T {
	case OpenPacket:
		switch val := packet.D.(type) {
		case HandshakeV3:
			if val.Upgrades == nil {
				val.Upgrades = []string{}
			}
			enc.w.Write(packet.T.Bytes())
			if err := json.NewEncoder(&stripLastNewlineWriter{enc.w}).Encode(val); err != nil {
				return ErrHandshakeEncode.F("v3", err)
			}
		default:
			return ErrInvalidHandshake.F("v3")
		}
	case BinaryPacket, MessagePacket:
		switch val := packet.D.(type) {
		case []byte: // binary data
			if _, err := enc.w.Write(append(packet.T.Bytes(), val...)); err != nil {
				return ErrPacketEncode.F("v3", err)
			}
			return nil
		}
		fallthrough // and encode MessagePacket with nil or string data like before...
	case PingPacket, PongPacket, ClosePacket, UpgradePacket, NoopPacket:
		if err := enc.PacketEncoderV2.Encode(PacketV2{Packet: packet.Packet}); err != nil {
			return ErrPacketEncode.F("v3", err)
		}
	default:
		return ErrInvalidPacketType.F(fmt.Sprintf("%T", packet.T))
	}

	return nil
}
