package protocol

import (
	"encoding/json"
	"io"
)

type (
	packetData       io.ReadWriter
	packetDataString struct{ x *string }
	packetDataArray  struct {
		marshalBinary   func(int, io.Reader) ([]byte, error)
		unmarshalBinary func([]byte, interface{}) error

		x []interface{}
	}
	packetDataObject struct {
		marshalBinary   func(int, io.Reader) ([]byte, error)
		unmarshalBinary func([]byte, interface{}) error

		x map[string]interface{}
	}

	packetDataObjectMarshal struct {
		marshalBinary func(int, io.Reader) ([]byte, error)

		x   map[string]interface{}
		num int
	}
)

//
// packetDataString
//

func (x *packetDataString) Read(p []byte) (n int, err error) {
	if x.x == nil || len(*x.x) == 0 {
		return
	}

	data, err := json.Marshal(x.x)
	n = copy(p, data)

	if n < len(data) { // this means there was no more room
		return n, ErrOnReadSoBuffer.BufferF("string data", data[n:], ErrShortRead)
	}

	return n, err
}

func (x *packetDataString) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	return len(p), json.Unmarshal(p, &x.x)
}

//
// packetDataArray
//

func (x *packetDataArray) Read(p []byte) (n int, err error) {
	if len(x.x) == 0 {
		// always return an error, because this could be empty or just a view of an empty array
		return 0, ErrOnReadSoBuffer.BufferF("binary data array", []byte("[]"), ErrEmptyDataArray)
	}

	n = copy(p, "[")
	nx, err := x.read(p[n:])
	n += nx

	return n, err
}

func (x *packetDataArray) read(p []byte) (n int, err error) {
	if len(x.x) == 0 {
		return
	}

	var nx, num int

	for j, val := range x.x {
		var data []byte

		switch v := val.(type) {
		case io.Reader:
			if x.marshalBinary == nil {
				err = ErrBinaryDataUnsupported
				break // from switch...
			}
			if data, err = x.marshalBinary(num, v); err != nil {
				return n, ErrBadBinaryMarshal.F(err).KV("array", "binary")
			}
			num++
		default:
			if data, err = json.Marshal(val); err != nil {
				return n, ErrBadMarshal.F(err).KV("array", "binary")
			}
		}

		var punct byte = ']'
		if j != len(x.x)-1 {
			punct = ','
		}

		data = append(data, punct)

		nx = copy(p[n:], data)
		n += nx

		if nx < len(data) {
			if err != nil {
				return n, ErrOnReadSoBuffer.BufferF("binary data array", data[nx:], err, ErrShortRead)
			}
			return n, ErrOnReadSoBuffer.BufferF("binary data array", data[nx:], ErrShortRead)
		}
	}

	return n, err
}

func (x *packetDataArray) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	if x.unmarshalBinary != nil {
		n, err = len(p), x.unmarshalBinary(p, &x.x)
		if err != nil {
			ErrBadUnmarshal.F(err).KV("array", "binary")
		}
	}

	return n, err
}

//
// packetDataObject
//

func (x *packetDataObject) Read(p []byte) (n int, err error) {
	if len(x.x) == 0 {
		// always return an error, because this could be empty or just a view of an empty array
		return 0, ErrOnReadSoBuffer.BufferF("binary data object", []byte("{}"), ErrEmptyDataArray)
	}

	data, err := json.Marshal(packetDataObjectMarshal{x: x.x, marshalBinary: x.marshalBinary})
	if err != nil {
		return n, ErrBadMarshal.F(err).KV("object", "binary")
	}

	n = copy(p, data)

	if n < len(data) {
		return n, ErrOnReadSoBuffer.BufferF("binary data object", data[n:], ErrShortRead) // PacketError{str: "buffer binary data object for read", buffer: data[n:], errs: []error{ErrShortRead}}
	}

	return n, err
}

func (x *packetDataObject) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	if x.unmarshalBinary != nil {
		n, err = len(p), x.unmarshalBinary(p, &x.x)
		if err != nil {
			ErrBadUnmarshal.F(err).KV("object", "binary")
		}
	}

	return n, err
}

func (o packetDataObjectMarshal) MarshalJSON() ([]byte, error) {
	var i int
	var out = []byte("{")

	for key, val := range o.x {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, `"`+key+`":`...)
		switch v := val.(type) {
		case io.Reader:
			if o.marshalBinary != nil {
				b, err := o.marshalBinary(o.num, v)
				if err != nil {
					return nil, err
				}
				out = append(out, b...)
			}
			o.num++
		case map[string]interface{}:
			b, err := json.Marshal(packetDataObjectMarshal{num: o.num, x: v, marshalBinary: o.marshalBinary})
			if err != nil {
				return nil, err
			}
			out = append(out, b...)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			out = append(out, b...)
		}
		i++
	}

	out = append(out, '}')
	return out, nil
}
