//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	rw "github.com/njones/socketio/internal/readwriter"
)

// PacketV4 is a v3 with different binary handling
type PacketV4 struct{ PacketV3 }

type PacketDecoderV4 struct{ *PacketDecoderV3 }

var NewPacketDecoderV4 _packetDecoderV4 = func(r io.Reader) *PacketDecoderV4 {
	return &PacketDecoderV4{PacketDecoderV3: &PacketDecoderV3{PacketDecoderV2: &PacketDecoderV2{read: rw.NewReader(r)}}}
}

func (dec *PacketDecoderV4) Decode(packet *PacketV4) error {
	if packet == nil {
		packet = &PacketV4{}
	}

	if packet.T == 0 && !packet.isOpenPacket {
		if dec.read.IsNotErr() && dec.read.ConditionalErr(dec.readPacketType(&packet.T, dec.read)).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v4")).IsNotErr() {
			packet.isOpenPacket = (packet.T == 0)
			defer func() { packet.isOpenPacket = false }() // always clear at the end...
		}
	}

	switch packet.T {
	case OpenPacket:
		var data HandshakeV4
		dec.read.SetDecoder(_packetJSONDecoder(json.NewDecoder)).Decode(&data).OnErrF(ErrDecodeHandshakeFailed, dec.read.Err())
		if dec.read.IsNotErr() {
			packet.D = &data
		}
		return dec.read.Err()
	case BinaryPacket:
		var data = new(bytes.Buffer)
		dec.read.SetDecoder(_packetBase64Decoder(base64.NewDecoder)).Decode(data).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v4"))
		packet.IsBinary = true
		packet.D = (io.Reader)(data)
		return dec.read.Err()
	}

	var v3 = packet.PacketV3
	if dec.read.IsNotErr() {
		dec.read.ConditionalErr(dec.PacketDecoderV3.Decode(&v3)).OnErrF(ErrDecodePacketFailed, dec.read.Err(), kv(ver, "v4"))
		packet.D = v3.D
	}

	return dec.read.Err()
}

type PacketEncoderV4 struct{ *PacketEncoderV3 }

var NewPacketEncoderV4 _packetEncoderV4 = func(w io.Writer) *PacketEncoderV4 {
	return &PacketEncoderV4{&PacketEncoderV3{PacketEncoderV2: &PacketEncoderV2{write: rw.NewWriter(w)}}}
}

func (enc *PacketEncoderV4) Encode(packet PacketV4) (err error) {
	switch packet.T {

	case OpenPacket:
		switch data := packet.D.(type) {
		case *HandshakeV4:
			if data.Upgrades == nil {
				data.Upgrades = []string{}
			}
			enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed)
			enc.write.UseEncoder(_packetJSONEncoder(json.NewEncoder)).Encode(data).OnErrF(ErrEncodeHandshakeFailed, enc.write.Err(), kv(ver, "v4"))
		default:
			return ErrUnexpectedHandshake.F("*HandshakeV4", data)
		}
		return enc.write.Err()
	case BinaryPacket:
		enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrEncodePacketFailed, enc.write.Err(), kv(ver, "v4"))
		switch data := packet.D.(type) {
		case []byte:
			enc.write.UseEncoder(_packetBase64encoder(base64.NewEncoder)).Encode(data).OnErrF(ErrEncodePacketFailed, enc.write.Err(), kv(ver, "v4"))
		case io.Reader:
			enc.write.UseEncoder(_packetBase64encoder(base64.NewEncoder)).Encode(data).OnErrF(ErrEncodePacketFailed, enc.write.Err(), kv(ver, "v4"))
		default:
			return fmt.Errorf("bad packet binary encode type: %T", data)
		}
		return enc.write.Err()
	}

	return enc.PacketEncoderV3.Encode(PacketV3{PacketV2{Packet: packet.Packet}, false})
}
