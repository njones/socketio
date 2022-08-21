package readwriter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func (wtr *Writer) UseEncoder(fn encWriter) *Writer {
	return &Writer{w: wtr.w, enc: fn}
}

func (wtr *Writer) SetEncoder(fn encWriter) *Writer {
	wtr.enc = fn
	return wtr
}

func (wtr *Writer) Encode(v interface{}) wtrErr {
	if wtr.enc != nil {
		wtr.err = wtr.enc.To(wtr.w).Encode(v)
		return onWtrErr{wtr}
	}
	switch val := v.(type) {
	case string:
		wtr.Write([]byte(val))
	case []byte:
		wtr.Write(val)
	case io.Reader:
		wtr.Copy(val)
	}
	return onWtrErr{wtr}
}

type (
	encWriter interface {
		To(io.Writer) encEncode
	}
	encEncode interface{ Encode(interface{}) error }
)

type JSONEncoder func(io.Writer) *json.Encoder

func (fn JSONEncoder) To(w io.Writer) encEncode {
	return fn(w)
}

type JSONEncoderStripNewline func(io.Writer) *json.Encoder

func (fn JSONEncoderStripNewline) To(w io.Writer) encEncode {
	return fn(&stripLastNewlineWriter{w})
}

type Base64Encoder func(*base64.Encoding, io.Writer) io.WriteCloser

func (fn Base64Encoder) To(w io.Writer) encEncode {
	return b64e{w: fn(base64.StdEncoding, w)}
}

type b64e struct{ w io.WriteCloser }

func (be b64e) Encode(v interface{}) error {
	switch val := v.(type) {
	case string:
		v = strings.NewReader(val)
	case []byte:
		v = bytes.NewReader(val)
	case io.Reader:
		v = val
	case io.WriterTo:
		_, err := val.WriteTo(be.w)
		if err == nil {
			be.w.Close()
		}
		return err
	default:
		return fmt.Errorf("val is not an io.Reader")
	}
	_, err := io.Copy(be.w, v.(io.Reader))
	if err == nil {
		err = be.w.Close()
	}
	return err
}

// This is because of https://golang.org/src/encoding/json/stream.go?s=4272:4319#L173
type stripLastNewlineWriter struct{ w io.Writer }

func (snw *stripLastNewlineWriter) Write(p []byte) (n int, err error) {
	if n = len(p) - 1; p[n] != '\n' { // assumes this is not in the middle of a write...
		n++
	}
	return snw.w.Write(p[:n])
}
