package readwriter

import (
	"bufio"
	"io"

	errs "github.com/njones/socketio/internal/errors"
)

func (wtr *Writer) Bytes(p []byte) wtrErr {
	if wtr.err != nil {
		return wtr
	}

	_, wtr.err = wtr.w.Write(p)
	return onWtrErr{wtr}
}

func (wtr *Writer) Byte(p byte) wtrErr {
	if wtr.err != nil {
		return wtr
	}

	wtr.err = wtr.w.WriteByte(p)
	return onWtrErr{wtr}
}

func (wtr *Writer) String(str string) wtrErr {
	if wtr.err != nil {
		return wtr
	}

	return wtr.Bytes([]byte(str))
}

func (wtr *Writer) Error() string {
	return wtr.err.Error()
}

func (wtr *Writer) To(w io.WriterTo) wtrErr {
	if wtr.err != nil {
		return wtr
	}

	_, wtr.err = w.WriteTo(wtr.w)
	return onWtrErr{wtr}
}

func (wtr *Writer) Copy(src io.Reader) wtrErr {
	if wtr.err != nil {
		return wtr
	}

	_, wtr.err = io.Copy(wtr.w, src)
	return onWtrErr{wtr}
}

func (wtr *Writer) Multi(ww ...io.Writer) *Writer {
	if wtr.err != nil {
		return wtr
	}

	wtr.w = bufio.NewWriter(io.MultiWriter(append([]io.Writer{wtr.w}, ww...)...))
	return wtr
}

type onWtrErr struct{ *Writer }

func (e onWtrErr) OnErr(err error) {
	if e.err != nil {
		e.err = err
	}
}
func (e onWtrErr) OnErrF(err errs.StringF, v ...interface{}) {
	if e.err != nil {
		e.err = err.F(v...)
	}
}
