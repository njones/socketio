//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import (
	"bytes"
	"io"

	rw "github.com/njones/socketio/internal/readwriter"
)

// PacketV3 is defined: https://github.com/socketio/engine.io-protocol/tree/v3
type PacketV3 struct {
	Packet
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
		if dec.read.IsNotErr() && dec.read.ConditionalErr(dec.readPacketType(&packet.T, dec.read)).OnErrF(ErrPacketDecode, "v3", dec.read.Err()).IsNotErr() {
			packet.isOpenPacket = (packet.T == 0)
			defer func() { packet.isOpenPacket = false }() // always clear at the end...
		}
	}

	switch packet.T {
	case OpenPacket:
		var data HandshakeV3
		dec.read.Decoder(newJSONDecoder()).Decode(&data)
		packet.D = data
		return dec.read.Err()
	}

	var v2 = PacketV2{Packet: packet.Packet}
	if dec.read.IsNotErr() {
		dec.read.ConditionalErr(dec.PacketDecoderV2.Decode(&v2)).OnErrF(ErrPacketDecode, "v3", dec.read.Err())
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
		case HandshakeV3:
			if data.Upgrades == nil {
				data.Upgrades = []string{}
			}
			enc.write.Bytes(packet.T.Bytes()).OnErr(ErrPacketEncode)
			enc.write.Encoder(newJSONEncoder()).Encode(data).OnErrF(ErrPacketEncode, "v3", enc.write.Err())
		default:
			return ErrInvalidHandshake.F("v3")
		}
		return enc.write.Err()
	}

	return enc.PacketEncoderV2.Encode(PacketV2{Packet: packet.Packet})
}
