//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import (
	"bytes"
	"encoding/json"
	"io"

	rw "github.com/njones/socketio/internal/readwriter"
)

// PacketV3 is defined: https://github.com/socketio/engine.io-protocol/tree/v3
type PacketV3 struct {
	PacketV2
	IsBinary bool
}

type PacketDecoderV3 struct{ *PacketDecoderV2 }

var NewPacketDecoderV3 _packetDecoderV3 = func(r io.Reader) *PacketDecoderV3 {
	return &PacketDecoderV3{PacketDecoderV2: &PacketDecoderV2{read: rw.NewReader(r)}}
}

func (dec *PacketDecoderV3) Decode(packet *PacketV3) error {
	if packet == nil {
		packet = &PacketV3{}
	}

	if packet.T == 0 && !packet.isOpenPacket {
		if dec.read.IsNotErr() && dec.read.ConditionalErr(dec.readPacketType(&packet.T, dec.read)).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v3")).IsNotErr() {
			packet.isOpenPacket = (packet.T == 0)
			defer func() { packet.isOpenPacket = false }() // always clear at the end...
		}
	}

	switch packet.T {
	case OpenPacket:
		var data HandshakeV3
		dec.read.SetDecoder(_packetJSONDecoder(json.NewDecoder)).Decode(&data).OnErrF(ErrDecodeHandshakeFailed, dec.read.Err(), kv(ver, "v3"))
		if dec.read.IsNotErr() {
			packet.D = &data
		}
		return dec.read.Err()
	}

	var v2 = packet.PacketV2
	if dec.read.IsNotErr() {
		dec.read.ConditionalErr(dec.PacketDecoderV2.Decode(&v2)).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v3"))
		packet.D = v2.D
		if packet.T == MessagePacket && packet.IsBinary {
			switch data := packet.D.(type) {
			case string:
				packet.D = bytes.NewReader([]byte(data))
			}
		}
	}
	return dec.read.Err()
}

type PacketEncoderV3 struct{ *PacketEncoderV2 }

var NewPacketEncoderV3 _packetEncoderV3 = func(w io.Writer) *PacketEncoderV3 {
	return &PacketEncoderV3{PacketEncoderV2: &PacketEncoderV2{write: rw.NewWriter(w)}}
}

func (enc *PacketEncoderV3) Encode(packet PacketV3) (err error) {

	switch packet.T {
	case OpenPacket:
		switch data := packet.D.(type) {
		case *HandshakeV3:
			if data.Upgrades == nil {
				data.Upgrades = []string{}
			}
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed)
			enc.write.UseEncoder(_packetJSONEncoder(json.NewEncoder)).Encode(data).OnErrF(ErrEncodeHandshakeFailed, enc.write, kv(ver, "v3"))
		default:
			return ErrUnexpectedHandshake.F("*HandshakeV3", data)
		}
		return enc.write.Err()
	}

	return enc.PacketEncoderV2.Encode(PacketV2{Packet: packet.Packet})
}
