//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import (
	"io"
)

const BinaryPacket PacketType = 255

type isBinaryPacket struct{ D interface{} }

func (pac *PacketV3) PacketRef() *Packet {
	if pac.IsBinary {
		pac.Packet.D = isBinaryPacket{D: pac.Packet.D}
	}
	return &pac.Packet
}

type _packetDecoderV3 func(r io.Reader) *PacketDecoderV3
type _packetEncoderV3 func(w io.Writer) *PacketEncoderV3
type _packetReaderV3 func(packet *PacketV3) (err error)
type _packetWriterV3 func(packet PacketV3) (err error)

func (pac _packetDecoderV3) From(r io.Reader) PacketReader { return _packetReaderV3(pac(r).Decode) }
func (pac _packetEncoderV3) To(w io.Writer) PacketWriter   { return _packetWriterV3(pac(w).Encode) }
func (pac _packetReaderV3) ReadPacket(packet PacketRef) (err error) {
	data := packet.PacketRef()
	var v = PacketV3{PacketV2{Packet: *data}, false}
	_, v.IsBinary = data.D.(isBinaryPacket)
	err = pac(&v)
	*packet.PacketRef() = v.Packet
	return err
}
func (pac _packetWriterV3) WritePacket(packet PacketVal) error {
	return pac(PacketV3{PacketV2{Packet: packet.PacketVal()}, false})
}
