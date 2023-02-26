package readwriter

import (
	"bufio"
	"errors"
	"io"

	errs "github.com/njones/socketio/internal/errors"
)

type rdrErr interface {
	OnErr(error)
	OnErrF(errs.StringF, ...interface{})
}

type rdrCondErr interface {
	rdrCondBool
	OnErr(error) rdrCondBool
	OnErrF(errs.StringF, ...interface{}) rdrCondBool
}

type rdrCondBool interface {
	IsErr() bool
	IsNotErr() bool
}

type Reader struct {
	r   *bufio.Reader
	dec decodeReader
	err error
}

func (rdr *Reader) Bufio() *bufio.Reader { return rdr.r }
func (rdr *Reader) SetErr(err error)     { rdr.err = err }

func (rdr *Reader) Err() error { return rdr.err }
func (rdr *Reader) ConvertErr(from, to error) *Reader {
	if errors.Is(rdr.err, from) {
		rdr.err = to
	}
	return rdr
}
func (rdr *Reader) OnErr(error)                         {}
func (rdr *Reader) OnErrF(errs.StringF, ...interface{}) {}
func (rdr *Reader) IsErr() bool                         { return rdr.err != nil }
func (rdr *Reader) IsNotErr() bool                      { return rdr.err == nil }
func (rdr *Reader) Read(p []byte) (n int, err error)    { return rdr.r.Read(p) }

type readerCond struct{ *Reader }

func (rdr readerCond) OnErr(error) rdrCondBool                         { return rdr }
func (rdr readerCond) OnErrF(errs.StringF, ...interface{}) rdrCondBool { return rdr }

func NewReader(r io.Reader) *Reader { return &Reader{r: bufio.NewReader(r)} }
