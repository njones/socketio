package readwriter

import (
	"io"

	errs "github.com/njones/socketio/internal/errors"
)

func (rdr *Reader) ConditionalErr(err error) rdrCondErr {
	if rdr.err != nil {
		return readerCond{rdr}
	}

	rdr.err = err
	return condErr{onRdrErr{rdr}}
}

func (rdr *Reader) Copy(w io.Writer) rdrErr {
	if rdr.err != nil {
		return rdr
	}

	_, rdr.err = io.Copy(w, rdr.r)
	return onRdrErr{rdr}
}

func (rdr *Reader) CopyN(w io.Writer, n int64) rdrErr {
	if rdr.err != nil {
		return rdr
	}

	_, rdr.err = io.CopyN(w, rdr.r, n)
	return onRdrErr{rdr}
}

type onRdrErr struct{ *Reader }

func (e onRdrErr) OnErr(err error) {
	if e.err != nil {
		e.err = err
	}
}
func (e onRdrErr) OnErrF(err errs.StringF, v ...interface{}) {
	if e.err != nil {
		e.err = err.F(v...)
	}
}

type condErr struct{ onRdrErr }

func (e condErr) OnErr(err error) rdrCondBool {
	e.onRdrErr.OnErr(err)
	return e
}

func (e condErr) OnErrF(err errs.StringF, v ...interface{}) rdrCondBool {
	e.onRdrErr.OnErrF(err, v...)
	return e
}

func (e condErr) IsErr() bool    { return e.err != nil }
func (e condErr) IsNotErr() bool { return e.err == nil }
