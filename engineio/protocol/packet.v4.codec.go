//go:build gc || eio_pac_v4
// +build gc eio_pac_v4

package protocol

import (
	"io"
)

type _packetDecoderV4 func(r io.Reader) *PacketDecoderV4
type _packetEncoderV4 func(w io.Writer) *PacketEncoderV4
type _packetReaderV4 func(packet *PacketV4) (err error)
type _packetWriterV4 func(packet PacketV4) (err error)

func (pac _packetDecoderV4) From(r io.Reader) PacketReader { return _packetReaderV4(pac(r).Decode) }
func (pac _packetEncoderV4) To(w io.Writer) PacketWriter   { return _packetWriterV4(pac(w).Encode) }
func (pac _packetReaderV4) ReadPacket(packet PacketRef) (err error) {
	data := packet.PacketRef()
	var v = PacketV4{PacketV3{PacketV2{Packet: *data}, false}}
	_, v.IsBinary = data.D.(isBinaryPacket)
	err = pac(&v)
	*packet.PacketRef() = v.Packet
	return err
}
func (pac _packetWriterV4) WritePacket(packet PacketVal) error {
	return pac(PacketV4{PacketV3: PacketV3{PacketV2{Packet: packet.PacketVal()}, false}})
}
