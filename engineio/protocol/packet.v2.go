//go:build gc || (eio_pac_v2 && eio_pac_v3)
// +build gc eio_pac_v2,eio_pac_v3

package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	rw "github.com/njones/socketio/internal/readwriter"
)

type PacketV2 struct{ Packet }

type PacketDecoderV2 struct {
	read *rw.Reader
}

var NewPacketDecoderV2 _packetDecoderV2 = func(r io.Reader) *PacketDecoderV2 {
	return &PacketDecoderV2{read: rw.NewReader(r)}
}

func (dec *PacketDecoderV2) Decode(packet *PacketV2) error {
	if packet == nil {
		packet = &PacketV2{}
	}

	if packet.T == 0 && !packet.isOpenPacket {
		if dec.read.IsNotErr() && dec.read.ConditionalErr(dec.readPacketType(&packet.T, dec.read)).IsNotErr() {
			packet.isOpenPacket = (packet.T == 0)
			defer func() { packet.isOpenPacket = false }() // always clear at the end...
		}
	}

	switch packetType := packet.T; packetType {
	case OpenPacket:
		var data HandshakeV2
		dec.read.SetDecoder(_packetJSONDecoder(json.NewDecoder)).Decode(&data).OnErrF(ErrDecodeHandshakeFailed, dec.read.Err(), kv(ver, "v2"))
		if dec.read.IsNotErr() {
			packet.D = &data
		}
	case MessagePacket:
		var data = new(strings.Builder)
		dec.read.Copy(data).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v2"))
		packet.D = data.String()
	case PingPacket, PongPacket:
		var data = new(strings.Builder)
		dec.read.Copy(data).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v2"))
		if data.Len() > 0 {
			packet.D = data.String()
		}
	case ClosePacket, UpgradePacket, NoopPacket:
		var data = new(strings.Builder)
		dec.read.Copy(data).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v2"))
	default:
		return fmt.Errorf("bad packet type: %T", packetType)
	}

	return dec.read.Err()
}

func (dec *PacketDecoderV2) readPacketType(packet io.Writer, r io.Reader) error {
	dec.read.CopyN(packet, 1).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v2"))
	return dec.read.Err()
}

type PacketEncoderV2 struct{ write *rw.Writer }

var NewPacketEncoderV2 _packetEncoderV2 = func(w io.Writer) *PacketEncoderV2 {
	return &PacketEncoderV2{write: rw.NewWriter(w)}
}

func (enc *PacketEncoderV2) Encode(packet PacketV2) (err error) {
	switch packet.T {
	case OpenPacket:
		switch data := packet.D.(type) {
		case *HandshakeV2: // must be a pointer so we can set our upgrades
			if data.Upgrades == nil {
				data.Upgrades = []string{}
			}
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodeHandshakeFailed, enc.write, kv(ver, "v2"))
			enc.write.UseEncoder(_packetJSONEncoder(json.NewEncoder)).Encode(data).OnErrF(ErrEncodeHandshakeFailed, enc.write, kv(ver, "v2"))
		default:
			return ErrUnexpectedHandshake.F("*HandshakeV2", data)
		}
	case MessagePacket, PingPacket, PongPacket:
		switch data := packet.D.(type) {
		case nil:
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
		case string:
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
			enc.write.Encode(data).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
		case []byte:
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
			enc.write.Encode(data).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
		case io.WriterTo:
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
			enc.write.Encode(data).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
		default:
			return ErrUnexpectedPacketData.F(data)
		}
	case ClosePacket, UpgradePacket, NoopPacket:
		enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed, enc.write, kv(ver, "v2"))
	default:
		return ErrUnexpectedPacketType.F(packet.T)
	}

	return enc.write.Err()
}
