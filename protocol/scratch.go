package protocol

import (
	"errors"
	"io"
)

type (
	stateFn      func(*scratch) stateFn // keeping state around the "scratch" pad data
	readStateFn  func([]byte) stateFn   // the read state
	writeStateFn func([]byte) stateFn   // the write state
)

// scratch holds the states the buffer and any data that's needed to facilitate reading
// and writing data to an underlining Packet object.
type scratch struct {
	data struct {
		set func(packetData)
	}

	read struct {
		n   int   // the current bytes read
		err error // the current error if not nil

		hasNamespaceComma bool

		buffer []byte        // the buffer to drain if not empty
		states []readStateFn // the remaining states to execute if not empty
	}
	write struct {
		n   int   // the current bytes written
		err error // the current error if not nil

		isBinary bool

		buffer []byte         // the buffer to drain if not empty
		states []writeStateFn // the remaining states to execute if not empty
	}
}

// resetRead resets the Reader to start again, reseting all of values so that
// it can start from a clean state
func (scr *scratch) resetRead() {
	scr.read.n, scr.read.err = 0, nil
	scr.read.hasNamespaceComma = false
}

// resetWrite resets the Writer to start again, reseting all of values so that
// it can start from a clean state
func (scr *scratch) resetWrite() {
	scr.write.n, scr.write.err = 0, nil
	scr.write.isBinary = false
}

// readFromPacket is the read state for reading an individual packet type, which
// is split out by field in the packet object. This executes a single state
// and removes it from the states list after execution.
func readFromPacket(r io.Reader) readStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			n, err := r.Read(p)

			scr.read.n += n
			scr.read.err = err

			if errors.As(err, &PacketError{}) && errors.Is(err, ErrShortRead) {
				scr.read.buffer = err.(PacketError).buffer
				scr.read.err = io.ErrUnexpectedEOF
			}

			scr.read.states = scr.read.states[1:]

			if err != nil || len(scr.read.states) == 0 {
				return nil
			}

			return scr.read.states[0](p[n:])
		}
	}
}

// readNamespaceFromPacket is a specialized state that wraps reading the Namespace
// packet, it checks to see if ut should add a "," before the namespace if there
// is a namespace to output.
func readNamespaceFromPacket(ns packetNS, ackID packetAckID, data packetData) readStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if len(ns) > 1 && (ackID > 0 || data != nil) {
				if ns[len(ns)-1] != ',' {
					ns += "," // not propgated because it's not a pointer
				}
			}
			return readFromPacket(ns)(p)
		}
	}
}

// applyAttachments checks to see if there are any binary streams to attach while reading
// the packet data.
func applyAttachments(data packetData, in *binaryStreamIn, out *binaryStreamOut) readStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {

			if array, ok := data.(*packetDataArray); ok {
				var num int
				for _, item := range array.x {
					if r, ok := item.(io.Reader); ok {
						out.rdr = append(out.rdr, r)
						num++
					}
				}
				*in = make(binaryStreamIn, num)
			}

			return readFromPacket(in)(p)
		}
	}
}

// readDataFromPacket wraps the packet type reader and can handle short reads
// it will populate the buffer and send back a <nil> error so that it can
// collect more data after the short read,
func readDataFromPacket(r io.Reader) readStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if r == nil {
				scr.read.states = scr.read.states[1:]
				if len(scr.read.states) == 0 {
					return nil
				}

				return scr.read.states[0](p)
			}

			next := readFromPacket(r)(p)(scr)

			if errors.As(scr.read.err, &PacketError{}) &&
				errors.Is(scr.read.err, ErrEmptyDataArray) {
				scr.read.buffer = scr.read.err.(PacketError).buffer
				scr.read.err = nil
			}

			return next
		}
	}
}

func writeToPacket(w io.Writer) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			n, err := w.Write(p)

			scr.write.n += n
			scr.write.err = err

			if errors.Is(err, ErrShortWrite) {
				scr.write.err = io.ErrUnexpectedEOF
			}

			if errors.Is(err, ErrUnexpectedJSONEnd) {
				scr.write.buffer = p
				scr.write.err = io.ErrUnexpectedEOF
			}

			if err != nil {
				return nil
			}

			scr.write.states = scr.write.states[1:]
			if len(scr.write.states) == 0 {
				return nil
			}

			return scr.write.states[0](p[n:])
		}
	}
}

func writeDataStringToPacket(w io.Writer) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if w == nil {
				w = &packetDataString{}
				defer func() {
					scr.data.set(packetData(w.(io.ReadWriter)))
				}()
			}

			// apply the full state before moving on...
			// so we can use the defer to add the data
			// back to the object
			return writeToPacket(w)(p)(scr)
		}
	}
}

func writeDataArrayToPacket(w io.Writer, incoming *binaryStreamIn) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if w == nil {
				w = &packetDataArray{}
				defer func() {
					scr.data.set(packetData(w.(io.ReadWriter)))
				}()
			}

			n, err := w.Write(p)

			scr.write.n += n
			scr.write.err = err

			if errors.Is(err, ErrUnexpectedJSONEnd) {
				scr.write.buffer = p

				scr.write.err = io.ErrUnexpectedEOF
			}

			if err != nil {
				return nil
			}

			// replace your binary data...
			data, _ := w.(*packetDataArray)
			for i, v := range data.x {
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
						data.x[i] = io.Reader(pr)
					}
				}
			}

			scr.write.states = scr.write.states[1:]
			if len(scr.write.states) == 0 {
				return nil
			}

			return scr.write.states[0](p[n:])
		}
	}
}

func writeDataObjectToPacket(w io.Writer, incoming *binaryStreamIn) writeStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if w == nil {
				w = &packetDataObject{}
				defer func() {
					scr.data.set(packetData(w.(io.ReadWriter)))
				}()
			}

			n, err := w.Write(p)

			scr.write.n += n
			scr.write.err = err

			if errors.Is(err, ErrUnexpectedJSONEnd) {
				scr.write.buffer = p

				scr.write.err = io.ErrUnexpectedEOF
			}

			if err != nil {
				return nil
			}

			// replace your binary data...
			data, _ := w.(*packetDataObject)
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
							data.x[i] = io.Reader(pr)
						} else {
							loop(m)
						}
					}
				}
			}
			loop(data.x)

			scr.write.states = scr.write.states[1:]
			if len(scr.write.states) == 0 {
				return nil
			}

			return scr.write.states[0](p[n:])
		}
	}
}

func withPacketData(v interface{}) packetData {
	switch val := v.(type) {
	case string:
		return &packetDataString{x: &val}
	case []interface{}:
		return &packetDataArray{x: val}
	case io.ReadWriter:
		return val
	default:
		return readWriteErr{ErrInvalidPacketType.F(val)}
	}
}
