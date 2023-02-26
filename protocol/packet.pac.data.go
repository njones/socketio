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

func (x packetDataString) Len() int {
	if x.x == nil {
		return 0
	}
	return len(*x.x) + 2 // the +2 is for "" quote marks
}

func (x *packetDataString) Read(p []byte) (n int, err error) {
	if x.x == nil || len(*x.x) == 0 {
		return
	}

	data, err := json.Marshal(x.x)
	n = copy(p, data)

	if n < len(data) { // this means there was no more room
		return n, ErrReadUseBuffer.BufferF("string data", data[n:], ErrShortRead)
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

func (x *packetDataArray) Len() (n int) {
	if len(x.x) == 0 {
		return 0
	}

	for _, v := range x.x {
		switch data := v.(type) {
		case interface{ Len() int }:
			n += data.Len()
		case string:
			n += len(data) + 2
		case int, int8, int16, int32, int64:
			n += intLen(data)
		case uint, uint8, uint16, uint32, uint64:
			n += intLen(data)
		case float32, float64:
			n += intLen(data)
		}
	}

	n += len(x.x) - 1 // adding in commas
	n += 2            // the 2 is: `[]`
	return n
}

func (x *packetDataArray) Read(p []byte) (n int, err error) {
	if len(x.x) == 0 {
		// always return an error, because this could be empty or just a view of an empty array
		return 0, ErrReadUseBuffer.BufferF("binary data array", []byte("[]"), ErrEmptyDataArray)
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
				return n, ErrMarshalBinaryDataFailed.F(err).KV("array", "binary")
			}
			num++
		default:
			if data, err = json.Marshal(val); err != nil {
				return n, ErrMarshalDataFailed.F(err).KV("array", "binary")
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
				return n, ErrReadUseBuffer.BufferF("binary data array", data[nx:], err, ErrShortRead)
			}
			return n, ErrReadUseBuffer.BufferF("binary data array", data[nx:], ErrShortRead)
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
			ErrUnmarshalDataFailed.F(err).KV("array", "binary")
		}
	}

	return n, err
}

//
// packetDataObject
//

func (x packetDataObject) Len() (n int) {
	if len(x.x) == 0 {
		return 0
	}

	for k, v := range x.x {
		n += len(k) + 3 // the +3 is: `"":`
		switch data := v.(type) {
		case interface{ Len() int }:
			n += data.Len()
		case string:
			n += len(data) + 2 // the +2 is `""`
		case int, int8, int16, int32, int64:
			n += intLen(data)
		case uint, uint8, uint16, uint32, uint64:
			n += intLen(data)
		case float32, float64:
			n += intLen(data)
		}
	}

	n += len(x.x) - 1 // adding in commas
	n += 2            // the 2 is: `{}`
	return n
}

func (x *packetDataObject) Read(p []byte) (n int, err error) {
	if len(x.x) == 0 {
		// always return an error, because this could be empty or just a view of an empty array
		return 0, ErrReadUseBuffer.BufferF("binary data object", []byte("{}"), ErrEmptyDataArray)
	}

	data, err := json.Marshal(packetDataObjectMarshal{x: x.x, marshalBinary: x.marshalBinary})
	if err != nil {
		return n, ErrMarshalDataFailed.F(err).KV("object", "binary")
	}

	n = copy(p, data)

	if n < len(data) {
		return n, ErrReadUseBuffer.BufferF("binary data object", data[n:], ErrShortRead) // PacketError{str: "buffer binary data object for read", buffer: data[n:], errs: []error{ErrShortRead}}
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
			ErrUnmarshalDataFailed.F(err).KV("object", "binary")
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
