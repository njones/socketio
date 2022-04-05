//go:build gc || (eio_pac_v2 && eio_pac_v3)
// +build gc eio_pac_v2,eio_pac_v3

package protocol

import (
	"io"
)

type _packetEncoderV2 func(w io.Writer) *PacketEncoderV2
type _packetDecoderV2 func(r io.Reader) *PacketDecoderV2
type _packetWriterV2 func(packet PacketV2) (err error)
type _packetReaderV2 func(packet *PacketV2) (err error)

func (pac _packetEncoderV2) To(w io.Writer) PacketWriter   { return _packetWriterV2(pac(w).Encode) }
func (pac _packetDecoderV2) From(r io.Reader) PacketReader { return _packetReaderV2(pac(r).Decode) }
func (pac _packetWriterV2) WritePacket(packet PacketVal) error {
	return pac(PacketV2{Packet: packet.PacketVal()})
}
func (pac _packetReaderV2) ReadPacket(packet PacketRef) error {
	var err error
	var v PacketV2
	if err = pac(&v); err != nil {
		return err
	}
	*packet.PacketRef() = v.Packet
	return err
}
