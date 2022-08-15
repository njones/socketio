package protocol

// • Remove the implicit connection to the default namespace
// In previous versions, a client was always connected to the default
// namespace, even if it requested access to another namespace.
//
// This is not the case anymore, the client must send a CONNECT packet in any case.
//
// Commits: 09b6f23 (server) and 249e0be (client)
//
// • rename ERROR to CONNECT_ERROR
// The meaning and the code number (4) are not modified: this packet type
// is still used by the server when the connection to a namespace is
// refused. But we feel the name is more self-descriptive.
//
// Commits: d16c035 (server) and 13e1db7c (client).
//
// the CONNECT packet now can contain a payload
// The client can send a payload for authentication/authorization purposes.
// This change means that the ID of the Socket.IO connection will now be
// different from the ID of the underlying Engine.IO connection (the one
// that is found in the query parameters of the HTTP requests).
//
// Commits: 2875d2c (server) and bbe94ad (client)
//
// • the payload CONNECT_ERROR packet is now an object instead of a plain string
// Commits: 54bf4a4 (server) and 0939395 (client)

import (
	"io"
)

const (
	ConnectErrorPacket packetType = ErrorPacket
)

var _ Packet = &PacketV5{}

type PacketV5 struct {
	packet
	packetBinary

	scratch `json:"-"` // holds buffers and such for writing out the wire format
}

func NewPacketV5() Packet {
	pac := &PacketV5{}
	pac.init()
	return pac
}

func (pac *PacketV5) init() {
	pac.packet.ket = func() Packet { return pac }
}

//
// provides the io.ReaderFrom/io.WriterTo interface for writing data
// to the underlining engineio packet
//

func (pac *PacketV5) Copy(w io.Writer, r io.Reader) (n int64, err error) {
	return io.Copy(underlining(w, r))
}
func (pac *PacketV5) ReadFrom(r io.Reader) (n int64, err error) { return pac.Copy(pac, r) }
func (pac *PacketV5) WriteTo(w io.Writer) (n int64, err error)  { return pac.Copy(w, pac) }

// provides the io.Reader/io.Writer interface for writing out the
// **version 3** socket.io wire string format

func (pac *PacketV5) Read(p []byte) (n int, err error) {
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

func (pac *PacketV5) Write(p []byte) (n int, err error) {
	if len(pac.scratch.write.states) == 0 &&
		len(pac.scratch.write.buffer) == 0 {

		pac.scratch.resetWrite()
		pac.scratch.write.states = []writeStateFn{
			writeToPacket(&pac.Type),
			binaryTypeCheckV5(&pac.Type),
			writeToPacket(&pac.incoming),
			writeToPacket(&pac.Namespace),
			writeToPacket(&pac.AckID),
			writeDataToPacketV5(
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

func (pac *PacketV5) ReadBinary() (bin func(r io.Reader) error) {
	if len(pac.incoming) == 0 {
		return nil
	}
	bin, pac.incoming = pac.incoming[0], pac.incoming[1:]
	return bin
}

func binaryTypeCheckV5(_type *packetType) writeStateFn {
	return binaryTypeCheckV4(_type)
}

func writeDataToPacketV5(w io.Writer, in *binaryStreamIn) writeStateFn {
	return writeDataToPacketV4(w, in)
}
