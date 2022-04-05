//go:build gc || (eio_pac_v2 && eio_pac_v3)
// +build gc eio_pac_v2,eio_pac_v3

package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type PacketV2 struct{ Packet }

type PacketDecoderV2 struct{ r io.Reader }

var NewPacketDecoderV2 _packetDecoderV2 = func(r io.Reader) *PacketDecoderV2 { return &PacketDecoderV2{r: r} }

func (dec *PacketDecoderV2) Decode(packet *PacketV2) error {
	if packet == nil {
		packet = &PacketV2{}
	}

	// check if packet is the default...
	if packet.T == 0 {
		if _, err := io.CopyN(&packet.T, dec.r, 1); err != nil {
			return ErrPacketDecode.F("v2", err)
		}
	}

	switch packet.T {
	case OpenPacket:
		var data HandshakeV2
		if err := json.NewDecoder(dec.r).Decode(&data); err != nil {
			return ErrHandshakeDecode.F("v2", err)
		}
		packet.D = data
	case MessagePacket:
		var data, err = io.ReadAll(dec.r)
		if err != nil {
			return ErrPacketDecode.F("v2", err)
		}
		packet.D = string(data)
	case PingPacket, PongPacket:
		var data = new(strings.Builder)
		if _, err := io.Copy(data, dec.r); err != nil {
			return ErrPacketDecode.F("v2", err)
		}
		if data.Len() > 0 {
			packet.D = data.String()
		}
	}

	return nil
}

type PacketEncoderV2 struct{ w io.Writer }

var NewPacketEncoderV2 _packetEncoderV2 = func(w io.Writer) *PacketEncoderV2 { return &PacketEncoderV2{w: w} }

func (enc *PacketEncoderV2) Encode(packet PacketV2) (err error) {
	switch packet.T {
	case OpenPacket:
		switch val := packet.D.(type) {
		case *HandshakeV2:
			if val.Upgrades == nil {
				val.Upgrades = []string{}
			}
			if _, err := enc.w.Write(packet.T.Bytes()); err != nil {
				return ErrHandshakeEncode.F("v2", err)
			}
			if err := json.NewEncoder(&stripLastNewlineWriter{enc.w}).Encode(val); err != nil {
				return ErrHandshakeEncode.F("v2", err)
			}
		default:
			return ErrInvalidHandshake.F("v2")
		}
	case MessagePacket, PingPacket, PongPacket:
		switch val := packet.D.(type) {
		case nil:
			if _, err := enc.w.Write(packet.T.Bytes()); err != nil {
				return ErrPacketEncode.F("v2", err)
			}
		case string:
			if _, err := enc.w.Write(append(packet.T.Bytes(), val...)); err != nil {
				return ErrPacketEncode.F("v2", err)
			}
		case io.WriterTo:
			if _, err := enc.w.Write(packet.T.Bytes()); err != nil {
				return ErrPacketEncode.F("v2", err)
			}
			if _, err := val.WriteTo(enc.w); err != nil {
				return ErrPacketEncode.F("v2", err)
			}
		default:
			return ErrInvalidPacketData.F(fmt.Sprintf("unexpected data type of: %T", val))
		}
	case ClosePacket, UpgradePacket, NoopPacket:
		if _, err := enc.w.Write(packet.T.Bytes()); err != nil {
			return ErrPacketEncode.F("v2", err)
		}
	default:
		return ErrInvalidPacketType.F(fmt.Sprintf("unexpected type of %s", packet.T))
	}

	return nil
}
