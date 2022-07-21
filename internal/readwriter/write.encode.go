package readwriter

import (
	"encoding/base64"
	"io"
)

func (wtr *Writer) Encoder(fn encoderWriter) encoderEncode {
	return wtrEncode{wtr: wtr, encode: fn.To(wtr.w)}
}

type (
	fnEncode      func(interface{}) error
	encoderWriter interface {
		To(io.Writer) func(interface{}) error
	}
	encoderEncode interface{ Encode(interface{}) wtrErr }
)

type wtrEncode struct {
	wtr    *Writer
	encode fnEncode
}

func (enc wtrEncode) Encode(v interface{}) wtrErr {
	if enc.wtr.err != nil {
		return enc.wtr
	}

	enc.wtr.err = enc.encode(v)
	return onWtrErr{enc.wtr}
}

func (wtr *Writer) Base64(enc *base64.Encoding) interface {
	Copy(io.Reader) wtrErr
	Bytes([]byte) wtrErr
} {
	if wtr.err != nil {
		return wtr
	}

	b64Wtr := base64.NewEncoder(enc, wtr.w)
	return wtrB64{wtr: wtr, w: b64Wtr}
}

type wtrB64 struct {
	wtr *Writer
	w   io.WriteCloser
}

func (b64 wtrB64) Copy(src io.Reader) wtrErr {
	if b64.wtr.err != nil {
		return b64.wtr
	}

	_, b64.wtr.err = io.Copy(b64.w, src)
	if b64.wtr.err == nil {
		b64.wtr.err = b64.w.Close()
	}
	return onWtrErr{b64.wtr}
}

func (b64 wtrB64) Bytes(p []byte) wtrErr {
	if b64.wtr.err != nil {
		return b64.wtr
	}

	_, b64.wtr.err = b64.w.Write(p)
	return onWtrErr{b64.wtr}
}
