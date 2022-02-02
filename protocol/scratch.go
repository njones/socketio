package protocol

import (
	"errors"
	"io"
)

type (
	stateFn      func(*scratch) stateFn
	readStateFn  func([]byte) stateFn
	writeStateFn func([]byte) stateFn
)

type scratch struct {
	data struct {
		set func(packetData)
	}

	read struct {
		n   int
		err error

		hasNamespaceComma bool

		buffer []byte
		states []readStateFn
	}
	write struct {
		n   int
		err error

		isBinary bool

		buffer []byte
		states []writeStateFn
	}
}

func (scr *scratch) resetRead() {
	scr.read.n, scr.read.err = 0, nil
	scr.read.hasNamespaceComma = false
}

func (scr *scratch) resetWrite() {
	scr.write.n, scr.write.err = 0, nil
	scr.write.isBinary = false
}

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

func readNamespaceFromPacket(ns packetNS, ackID packetAckID, data packetData) readStateFn {
	return func(p []byte) stateFn {
		return func(scr *scratch) stateFn {
			if len(ns) > 1 && (ackID > 0 || data != nil) {
				if ns[len(ns)-1] != ',' {
					ns += "," // not propgated becuase it's not a pointer
				}
			}
			return readFromPacket(ns)(p)
		}
	}
}

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

	/*
		if rw, ok := v.(io.ReadWriter); ok {
			return rw
		}
	*/
}
