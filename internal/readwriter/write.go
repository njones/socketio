package readwriter

import (
	"bufio"
	"io"

	errs "github.com/njones/socketio/internal/errors"
)

type wtrErr interface {
	OnErr(error)
	OnErrF(errs.StringF, ...interface{})
}

type Writer struct {
	w   *bufio.Writer
	enc encWriter
	err error
}

func (wtr *Writer) Bufio() *bufio.Writer { return wtr.w }

func (wtr *Writer) Err() error {
	if wtr.err == nil {
		wtr.err = wtr.w.Flush()
	}
	return wtr.err
}
func (wtr *Writer) OnErr(error)                         {}
func (wtr *Writer) OnErrF(errs.StringF, ...interface{}) {}

func (wtr *Writer) Write(p []byte) (n int, err error) { return wtr.w.Write(p) }

func NewWriter(w io.Writer) *Writer {
	if ww, ok := w.(interface{ PropagateWriter() *Writer }); ok {
		return ww.PropagateWriter()
	}
	return &Writer{w: bufio.NewWriterSize(w, 1e6)}
}
