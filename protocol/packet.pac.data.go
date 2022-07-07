package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

type (
	packetData       io.ReadWriter
	packetDataString struct{ x *string }
	packetDataArray  struct {
		skipBinary bool

		x []interface{}
	}
	packetDataObject struct {
		skipBinary bool

		x map[string]interface{}
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

		switch val.(type) {
		case (io.Reader):
			// TODO(njones) for v2 create msgpack base64'd blob?
			if x.skipBinary {
				continue
			}
			data = []byte(fmt.Sprintf(`{"_placeholder":true,"num":%d}`, num))
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
			return n, ErrOnReadSoBuffer.BufferF("binary data array", data[nx:], ErrShortRead)
		}
	}

	return n, err
}

func (x *packetDataArray) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	n, err = len(p), json.Unmarshal(p, &x.x)
	if err != nil {
		ErrBadUnmarshal.F(err).KV("array", "binary")
	}
	// TODO(njones): for v2 loop thorugh base64/msgpack decode binary blobs
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

	data, err := json.Marshal(packetDataObjectJSON{m: x.x})
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

	n, err = len(p), json.Unmarshal(p, &x.x)
	if err != nil {
		ErrBadUnmarshal.F(err).KV("object", "binary")
	}

	// TODO(njones): for v2 loop through base64/msgpack decode binary blobs
	return n, err
}

type packetDataObjectJSON struct {
	skipBinary bool

	n int
	m map[string]interface{}
}

func (o packetDataObjectJSON) MarshalJSON() ([]byte, error) {
	var i int
	var out = []byte("{")

	for key, val := range o.m {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, `"`+key+`":`...)
		switch v := val.(type) {
		case io.Reader:
			if o.skipBinary {
				out = append(out, `{}`...)
				break
			}
			out = append(out, `{_placeholder":true,"num":`+strconv.Itoa(o.n)+`}`...)
			o.n++
		case map[string]interface{}:
			j, e := json.Marshal(packetDataObjectJSON{n: o.n, m: v})
			if e != nil {
				return nil, e
			}
			out = append(out, j...)
		default:
			j, e := json.Marshal(v)
			if e != nil {
				return nil, e
			}
			out = append(out, j...)
		}
		i++
	}

	out = append(out, '}')
	return out, nil
}
