package readwriter

import (
	"encoding/base64"
	"io"
)

func (rdr *Reader) Decoder(fn decoderReader) decoderDecode {
	return rdrDecode{rdr: rdr, decode: fn.From(rdr.r)}
}

type (
	fnDecode      func(interface{}) error
	decoderReader interface {
		From(io.Reader) func(interface{}) error
	}
	decoderDecode interface{ Decode(interface{}) rdrErr }
)

type rdrDecode struct {
	rdr    *Reader
	decode fnDecode
}

func (dec rdrDecode) Decode(v interface{}) rdrErr {
	if dec.rdr.err != nil {
		return dec.rdr
	}
	dec.rdr.err = dec.decode(v)
	return onRdrErr{dec.rdr}
}

func (rdr *Reader) Base64(enc *base64.Encoding) interface{ Copy(io.Writer) rdrErr } {
	if rdr.err != nil {
		return rdr
	}

	b64Rdr := base64.NewDecoder(enc, rdr.r)
	return rdrB64{rdr: rdr, r: b64Rdr}
}

type rdrB64 struct {
	rdr *Reader
	r   io.Reader
}

func (b64 rdrB64) Copy(dst io.Writer) rdrErr {
	if b64.rdr.err != nil {
		return b64.rdr
	}

	_, b64.rdr.err = io.Copy(dst, b64.r)
	return onRdrErr{b64.rdr}
}
