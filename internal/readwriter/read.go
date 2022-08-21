package readwriter

import (
	"bufio"
	"errors"
	"io"

	errs "github.com/njones/socketio/internal/errors"
)

type rdrErr interface {
	OnErr(errs.String)
	OnErrF(errs.String, ...interface{})
}

type rdrCondErr interface {
	rdrCondBool
	OnErr(errs.String) rdrCondBool
	OnErrF(errs.String, ...interface{}) rdrCondBool
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
func (rdr *Reader) OnErr(errs.String)                  {}
func (rdr *Reader) OnErrF(errs.String, ...interface{}) {}
func (rdr *Reader) IsErr() bool                        { return rdr.err != nil }
func (rdr *Reader) IsNotErr() bool                     { return rdr.err == nil }
func (rdr *Reader) Read(p []byte) (n int, err error)   { return rdr.r.Read(p) }

type readerCond struct{ *Reader }

func (rdr readerCond) OnErr(errs.String) rdrCondBool                  { return rdr }
func (rdr readerCond) OnErrF(errs.String, ...interface{}) rdrCondBool { return rdr }

func NewReader(r io.Reader) *Reader { return &Reader{r: bufio.NewReader(r)} }
