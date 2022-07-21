//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import (
	"bytes"
	"encoding/base64"
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
		if dec.read.IsNotErr() && dec.read.ConditionalErr(dec.readPacketType(&packet.T, dec.read)).OnErrF(ErrPacketDecode, "v4", dec.read.Err()).IsNotErr() {
			packet.isOpenPacket = (packet.T == 0)
			defer func() { packet.isOpenPacket = false }() // always clear at the end...
		}
	}

	switch packet.T {
	case BinaryPacket:
		var data = new(bytes.Buffer)
		dec.read.Base64(base64.StdEncoding).Copy(data).OnErrF(ErrPacketDecode, "v4", dec.read.Err())
		packet.IsBinary = true
		packet.D = (io.Reader)(data)
		return dec.read.Err()
	}

	var v3 = PacketV3{Packet: packet.Packet}
	if dec.read.IsNotErr() {
		dec.read.ConditionalErr(dec.PacketDecoderV3.Decode(&v3)).OnErrF(ErrPacketDecode, "v4", dec.read.Err())
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
	case BinaryPacket:
		enc.write.Bytes(packet.T.Bytes()).OnErrF(ErrPacketEncode, "v4", enc.write.Err())
		switch data := packet.D.(type) {
		case []byte:
			enc.write.Base64(base64.StdEncoding).Bytes(data).OnErrF(ErrPacketEncode, "v4", enc.write.Err())
		case io.Reader:
			enc.write.Base64(base64.StdEncoding).Copy(data).OnErrF(ErrPacketEncode, "v4", enc.write.Err())
		default:
			return fmt.Errorf("bad packet dinary encode type: %T", data)
		}
		return enc.write.Err()
	}

	return enc.PacketEncoderV3.Encode(PacketV3{Packet: packet.Packet})
}
