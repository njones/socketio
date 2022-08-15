package protocol

// • add a BINARY_ACK packet type
// Previously, an ACK packet was always treated as if
// it may contain binary objects, with recursive search
// for such objects, which could hurt performance.

import (
	"io"
)

const (
	BinaryAckPacket packetType = BinaryEventPacket + 1
)

var _ Packet = &PacketV4{}

type PacketV4 struct {
	packet
	packetBinary

	scratch `json:"-"` // holds buffers and such for writing out the wire format
}

func NewPacketV4() Packet {
	pac := &PacketV4{}
	pac.init()
	return pac
}

func (pac *PacketV4) init() {
	pac.packet.ket = func() Packet { return pac }
}

//
// provides the io.ReaderFrom/io.WriterTo interface for writing data
// to the underlining engineio packet
//

func (pac *PacketV4) Copy(w io.Writer, r io.Reader) (n int64, err error) {
	return io.Copy(underlining(w, r))
}
func (pac *PacketV4) ReadFrom(r io.Reader) (n int64, err error) { return pac.Copy(pac, r) }
func (pac *PacketV4) WriteTo(w io.Writer) (n int64, err error)  { return pac.Copy(w, pac) }

// provides the io.Reader/io.Writer interface for writing out the
// **version 3** socket.io wire string format

func (pac *PacketV4) Read(p []byte) (n int, err error) {
	if len(pac.scratch.read.states) == 0 &&
		len(pac.scratch.read.buffer) == 0 {

		pac.scratch.resetRead()
		pac.scratch.read.states = []readStateFn{
			readFromPacket(pac.Type),
			applyAttachments(
				pac.Data,
				&pac.incoming,
				&pac.outgoing,
			),
			readNamespaceFromPacket(
				pac.Namespace,
				pac.AckID,
				pac.Data,
			),
			readFromPacket(pac.AckID),
			readDataFromPacket(pac.Data),
		}
	}

	if len(pac.scratch.read.buffer) > 0 {
		n = copy(p, pac.scratch.read.buffer)
		pac.scratch.read.buffer = pac.scratch.read.buffer[n:]
		if len(pac.scratch.read.buffer) > 0 {
			return n, io.ErrUnexpectedEOF
		}
		pac.scratch.read.n = n
	}

	if len(pac.scratch.read.states) > 0 {
		var state = pac.scratch.read.states[0](p[n:])
		for state != nil {
			state = state(&pac.scratch)
		}
	}

	if len(pac.scratch.read.states) == 0 {
		if len(pac.scratch.read.buffer) == 0 {
			if pac.scratch.read.err == nil {
				pac.scratch.read.err = io.EOF // we are done with everything, so send io.EOF
			}
		}
	}

	return pac.scratch.read.n, pac.scratch.read.err
}

func (pac *PacketV4) Write(p []byte) (n int, err error) {
	if len(pac.scratch.write.states) == 0 &&
		len(pac.scratch.write.buffer) == 0 {

		pac.scratch.resetWrite()
		pac.scratch.write.states = []writeStateFn{
			writeToPacket(&pac.Type),
			binaryTypeCheckV4(&pac.Type),
			writeToPacket(&pac.incoming),
			writeToPacket(&pac.Namespace),
			writeToPacket(&pac.AckID),
			writeDataToPacketV4(
				pac.Data,
				&pac.incoming,
			),
		}
	}

	pac.scratch.data.set = func(d packetData) { pac.Data = d }

	if len(pac.scratch.write.buffer) > 0 {
		p = append(pac.scratch.write.buffer, p...)
		pac.scratch.write.buffer = pac.scratch.write.buffer[:0]
	}

	if len(pac.scratch.write.states) > 0 {
		var state = pac.scratch.write.states[0](p)
		for state != nil {
			state = state(&pac.scratch)
		}
	}

	return pac.scratch.write.n, pac.scratch.write.err
}

func (pac *PacketV4) ReadBinary() (bin func(r io.Reader) error) {
	if len(pac.incoming) == 0 {
		return nil
	}
	bin, pac.incoming = pac.incoming[0], pac.incoming[1:]
	return bin
}

func binaryTypeCheckV4(_type *packetType) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			next := binaryTypeCheckV2(_type)(p)(scr)

			if !scr.write.isBinary && _type.Byte() == BinaryAckPacket.Byte() {
				scr.write.isBinary = true
			}

			return next
		}
	}
}

func writeDataToPacketV4(w io.Writer, in *binaryStreamIn) writeStateFn {
	return writeDataToPacketV3(w, in)
}
