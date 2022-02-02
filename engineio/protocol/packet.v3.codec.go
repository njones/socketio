package protocol

import (
	"io"
)

type _packetEncoderV3 func(w io.Writer) *PacketEncoderV3
type _packetDecoderV3 func(r io.Reader) *PacketDecoderV3
type _packetWriterV3 func(packet PacketV3) (err error)
type _packetReaderV3 func(packet *PacketV3) (err error)

func (pac _packetEncoderV3) To(w io.Writer) PacketWriter   { return _packetWriterV3(pac(w).Encode) }
func (pac _packetDecoderV3) From(r io.Reader) PacketReader { return _packetReaderV3(pac(r).Decode) }
func (pac _packetWriterV3) WritePacket(packet PacketVal) error {
	return pac(PacketV3{Packet: packet.PacketVal()})
}
func (pac _packetReaderV3) ReadPacket(packet PacketRef) error {
	var err error
	var v PacketV3
	if err = pac(&v); err != nil {
		return err
	}
	*packet.PacketRef() = v.Packet
	return err
}
