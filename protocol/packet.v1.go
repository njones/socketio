package protocol

import (
	"encoding/json"
	"io"
)

// The packet type codes available in the socket.io protocol version 1
const (
	ConnectPacket packetType = iota
	DisconnectPacket
	EventPacket
	AckPacket
	ErrorPacket
)

// check that packet v1 is a valid Packet interface
var _ Packet = &PacketV1{}

// PacketV1 embeds a base packet and will convert the SocketIO version 1 values
type PacketV1 struct {
	packet

	scratch `json:"-,omitempty"` // holds buffers and such for writing out the wire format
}

// NewPacketV1 returns a Packet interface that will read/write socket.io version 1 packets
func NewPacketV1() Packet {
	pac := &PacketV1{}
	pac.init()
	return pac
}

func (pac *PacketV1) init() {
	pac.packet.ket = func() Packet { return pac }
}

func (pac *PacketV1) WithData(x interface{}) Packet {
	pac.packet.WithData(x)
	switch pac.packet.Data.(type) {
	case *packetDataArray:
		pac.packet.Data.(*packetDataArray).marshalBinary = nil
		pac.packet.Data.(*packetDataArray).unmarshalBinary = packetDataArrayUnmarshalV1
	case *packetDataObject:
		pac.packet.Data.(*packetDataObject).marshalBinary = packetDataObjectMarshalV1
		pac.packet.Data.(*packetDataObject).unmarshalBinary = packetDataObjectUnmarshalV1
	}
	return pac
}

//
// provides the io.ReaderFrom/io.WriterTo interface for writing data
// to the underlining engineio packet
//

// Copy forces an io.Copy to use the .Read and .Write methods to provide the copy
func (pac *PacketV1) Copy(w io.Writer, r io.Reader) (n int64, err error) {
	return io.Copy(underlining(w, r))
}

// ReadFrom copies the []bytes from the socket.io wire format to the PacketV1 struct.
func (pac *PacketV1) ReadFrom(r io.Reader) (n int64, err error) { return pac.Copy(pac, r) }

// WriteTo copies the PacketV1 struct to the []byte socket.io wire format.
func (pac *PacketV1) WriteTo(w io.Writer) (n int64, err error) { return pac.Copy(w, pac) }

// provides the io.Reader/io.Writer interface to
// read and write the *version 1* socket.io wire
// (string) format

// Read writes out the PacketV1 object to a socket.io protocol version 1 wire format
// to p []bytes. This method can handle Read being called multiple times during the
// course of populating the []bytes.
func (pac *PacketV1) Read(p []byte) (n int, err error) {
	if len(pac.scratch.read.states) == 0 &&
		len(pac.scratch.read.buffer) == 0 {

		pac.scratch.resetRead()
		pac.scratch.read.states = []readStateFn{
			readFromPacket(pac.Type),
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

// Write takes in protocol version 1 wire format in p []bytes. This method
// can handle Read being called multiple times during the course of populating the PacketV1 object.
func (pac *PacketV1) Write(p []byte) (n int, err error) {
	if len(pac.scratch.write.states) == 0 &&
		len(pac.scratch.write.buffer) == 0 {

		pac.scratch.resetWrite()
		pac.scratch.write.states = []writeStateFn{
			writeToPacket(&pac.Type),
			writeToPacket(&pac.Namespace),
			writeToPacket(&pac.AckID),
			writeDataToPacketV1(pac.Data),
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

// writeDataToPacketV1 takes in a writer w that will contain the
// Data portion of the socket.io protocol version 1 wire format
// and convert it to the proper internal data format for the
// PacketV1 object.
func writeDataToPacketV1(w io.Writer) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if len(p) == 0 {
				return nil
			}

			switch p[0] {
			case '"':
				return writeDataStringToPacket(w)(p)
			case '[':
				if w == nil {
					w = newPacketDataArray(withArrayMarshal(nil),
						withArrayUnmarshal(packetDataArrayUnmarshalV1))
					defer func() {
						scr.data.set(packetData(w.(io.ReadWriter)))
					}()
				}

				return writeToPacket(w)(p)(scr)

			case '{':
				if w == nil {
					w = newPacketDataObject(withObjectMarshal(packetDataObjectMarshalV1),
						withObjectUnmarshal(packetDataObjectUnmarshalV1))
					defer func() {
						scr.data.set(packetData(w.(io.ReadWriter)))
					}()
				}

				return writeToPacket(w)(p)(scr)
			}

			return nil
		}
	}
}

func packetDataArrayUnmarshalV1(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return ErrBinaryDataUnsupported
	}
	return nil
}

func packetDataObjectMarshalV1(_ int, v io.Reader) ([]byte, error) {
	return json.Marshal(v)
}

func packetDataObjectUnmarshalV1(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return ErrBinaryDataUnsupported
	}
	return nil
}
