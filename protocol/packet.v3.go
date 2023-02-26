package protocol

// • remove the usage of msgpack to encode packets
//   containing binary objects (see also 299849b)
//   https://github.com/socketio/socket.io-protocol#difference-between-v3-and-v2

import (
	"encoding/json"
	"io"
)

var _ Packet = &PacketV3{}

type PacketV3 struct {
	packet
	packetBinary

	scratch `json:"-"` // holds buffers and such for writing out the wire format
}

func NewPacketV3() Packet {
	pac := &PacketV3{}
	pac.init()
	return pac
}

func (pac *PacketV3) init() {
	pac.packet.ket = func() Packet { return pac }
}

// func (pac *PacketV3) WithData(x interface{}) Packet {

//
// provides the io.ReaderFrom/io.WriterTo interface for writing data
// to the underlining engineio packet
//

func (pac *PacketV3) Copy(w io.Writer, r io.Reader) (n int64, err error) {
	return io.Copy(underlining(w, r))
}
func (pac *PacketV3) ReadFrom(r io.Reader) (n int64, err error) { return pac.Copy(pac, r) }
func (pac *PacketV3) WriteTo(w io.Writer) (n int64, err error)  { return pac.Copy(w, pac) }

// provides the io.Reader/io.Writer interface for writing out the
// **version 3** socket.io wire string format

func (pac *PacketV3) Read(p []byte) (n int, err error) {
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

func (pac *PacketV3) Write(p []byte) (n int, err error) {
	if len(pac.scratch.write.states) == 0 &&
		len(pac.scratch.write.buffer) == 0 {

		pac.scratch.resetWrite()
		pac.scratch.write.states = []writeStateFn{
			writeToPacket(&pac.Type),
			binaryTypeCheckV3(&pac.Type),
			writeToPacket(&pac.incoming),
			writeToPacket(&pac.Namespace),
			writeToPacket(&pac.AckID),
			writeDataToPacketV3(
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

func binaryTypeCheckV3(_type *packetType) writeStateFn { return binaryTypeCheckV2(_type) }

func writeDataToPacketV3(w io.Writer, in *binaryStreamIn) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if len(p) == 0 {
				return nil
			}

			switch p[0] {
			case '"':
				return writeDataStringToPacket(w)(p)
			case '[':
				return writeDataArrayToPacket(w,
					withArrayUnmarshal(packetDataArrayUnmarshalV3(in)))(p)
			case '{':
				return writeDataObjectToPacket(w,
					withObjectUnmarshal(packetDataObjectUnmarshalV3(in)))(p)
			}

			return nil
		}
	}
}

type copyReader struct {
	r   io.Reader
	err chan error
}

func (cr copyReader) Read(p []byte) (n int, err error) {
	n, err = cr.r.Read(p)
	select {
	case e := <-cr.err:
		return n, e
	default:
		return n, err
	}
}

func packetDataArrayUnmarshalV3(incoming *binaryStreamIn) func([]byte, interface{}) error {
	return func(data_ []byte, vw interface{}) error {
		err := json.Unmarshal(data_, vw)
		if err != nil {
			return ErrUnmarshalInitialFieldFailed.F(err)
		}

		// replace your binary data...
		if datax, ok := vw.(*[]interface{}); ok {
			for i, v := range *datax {
				if m, ok := v.(map[string]interface{}); ok {
					if isPlaceholder, ok := m["_placeholder"].(bool); ok && isPlaceholder {
						pr, pw := io.Pipe()
						idx := int(m["num"].(float64))

						cr := copyReader{r: pr, err: make(chan error, 1)}

						(*incoming)[idx] = func(r io.Reader) error {

							go func() {
								_, err := io.Copy(pw, r)
								if err != nil {
									cr.err <- err
									return
								}
								cr.err <- pw.Close()
							}()
							return nil
						}

						vvw, _ := vw.(*[]interface{})
						(*vvw)[i] = io.Reader(cr)
					}
				}
			}
		}
		return nil
	}
}

func packetDataObjectUnmarshalV3(incoming *binaryStreamIn) func([]byte, interface{}) error {
	return func(data_ []byte, vw interface{}) error {
		err := json.Unmarshal(data_, vw)
		if err != nil {
			return ErrUnmarshalInitialFieldFailed.F(err)
		}

		// replace your binary data...
		data, _ := vw.(*map[string]interface{})
		var loop func(map[string]interface{})
		loop = func(x map[string]interface{}) {
			for i, v := range x {
				if m, ok := v.(map[string]interface{}); ok {
					if isPlaceholder, ok := m["_placeholder"].(bool); ok && isPlaceholder {
						pr, pw := io.Pipe()
						idx := int(m["num"].(float64))
						(*incoming)[idx] = func(r io.Reader) error {

							e := make(chan error, 1)
							go func() {
								io.Copy(pw, r)
								e <- pw.Close()
							}()

							return <-e
						}
						(*data)[i] = io.Reader(pr)
					} else {
						loop(m)
					}
				}
			}
		}
		loop(*data)

		return nil
	}
}
