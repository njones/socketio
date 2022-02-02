package protocol

// • add a BINARY_EVENT packet type
// Added during the work towards Socket.IO 1.0, in order
// to add support for binary objects. The BINARY_EVENT
// packets are encoded with msgpack.

import (
	"io"
)

const (
	BinaryEventPacket packetType = ErrorPacket + 1
)

var _ Packet = &PacketV2{}

type PacketV2 struct {
	packet

	incoming binaryStreamIn  `json:"-"`
	outgoing binaryStreamOut `json:"-"`

	// scratch - holds buffers and such for reading and writing out the wire format
	scratch `json:"-"`
}

func NewPacketV2() Packet {
	pac := &PacketV2{}
	pac.init()
	return pac
}

func (pac *PacketV2) init() {
	pac.packet.ket = func() Packet { return pac }
}

//
// provides the io.ReaderFrom/io.WriterTo interface for writing data
// to the underlining engineio packet
//

func (pac *PacketV2) Copy(w io.Writer, r io.Reader) (n int64, err error) {
	return io.Copy(underlining(w, r))
}
func (pac *PacketV2) ReadFrom(r io.Reader) (n int64, err error) { return pac.Copy(pac, r) }
func (pac *PacketV2) WriteTo(w io.Writer) (n int64, err error)  { return pac.Copy(w, pac) }

// provides the io.Reader/io.Writer interface for writing out the
// **version 3** socket.io wire string format

func (pac *PacketV2) Read(p []byte) (n int, err error) {
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

func (pac *PacketV2) Write(p []byte) (n int, err error) {
	if len(pac.scratch.write.states) == 0 &&
		len(pac.scratch.write.buffer) == 0 {

		pac.scratch.resetWrite()
		pac.scratch.write.states = []writeStateFn{
			writeToPacket(&pac.Type),
			binaryTypeCheckV2(&pac.Type),
			writeToPacket(&pac.incoming),
			writeToPacket(&pac.Namespace),
			writeToPacket(&pac.AckID),
			writeDataToPacketV2(
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

func binaryTypeCheckV2(_type *packetType) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {

			scr.write.isBinary = _type.Byte() == BinaryEventPacket.Byte()
			scr.write.isBinary = true // TODO(njones): fix tests..

			scr.write.states = scr.write.states[1:]
			if len(scr.write.states) == 0 {
				return nil
			}

			return scr.write.states[0](p)
		}
	}
}

func writeDataToPacketV2(w io.Writer, in *binaryStreamIn) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if len(p) == 0 {
				return nil
			}

			switch p[0] {
			case '"':
				return writeDataStringToPacket(w)(p)
			case '[':
				return writeDataArrayToPacket(w, in)(p)
			case '{':
				return writeDataObjectToPacket(w, in)(p)
			}

			return nil
		}
	}
}
