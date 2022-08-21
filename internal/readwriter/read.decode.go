package readwriter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

func (rdr *Reader) SetDecoder(fn decodeReader) *Reader {
	rdr.dec = fn
	return rdr
}

func (rdr *Reader) Decode(v interface{}) wtrErr {
	if rdr.dec != nil {
		rdr.err = rdr.dec.From(rdr.r).Decode(v)
		return onRdrErr{rdr}
	}
	switch val := v.(type) {
	case string:
		rdr.Read([]byte(val))
	case []byte:
		//rdr.Write(val)
	case io.Writer:
		rdr.Copy(val)
	default:
		_ = val
	}

	return onRdrErr{rdr}
}

type (
	decodeReader interface {
		From(io.Reader) decDecode
	}
	decDecode interface{ Decode(interface{}) error }
)

type JSONDecoder func(io.Reader) *json.Decoder

func (fn JSONDecoder) From(r io.Reader) decDecode {
	return fn(r)
}

type Base64Decoder func(*base64.Encoding, io.Reader) io.Reader

func (fn Base64Decoder) From(r io.Reader) decDecode {
	return b64d{r: fn(base64.StdEncoding, r)}
}

type b64d struct{ r io.Reader }

func (bd b64d) Decode(v interface{}) error {
	switch val := v.(type) {
	case io.Writer:
		_, err := io.Copy(val, bd.r)
		return err
	}
	return fmt.Errorf("val is not an io.Writer")
}
