package protocol

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
)

type recordScan struct {
	r, thru *bufio.Reader // holds the original buffer
	eor     bool
	buf     *bytes.Buffer

	// Because we need to send a (0, io.EOF) read to
	// end the .Read calls, otherwise bufio supresses
	// the io.EOF if n > 0 when it's sent. So we wait
	// for another read from a *bufio.Reader and send
	// the (0, io.EOF) it's expecting.
	isBufioReader bool
}

func (s *recordScan) Read(p []byte) (n int, err error) {
	if s.isBufioReader && s.eor {
		s.eor = false
		return 0, io.EOF
	}
	s.buf.ReadFrom(s.thru)
	limRdr := io.LimitReader(io.MultiReader(s.buf, s.r), int64(len(p)))
	s.thru = bufio.NewReader(limRdr)
	out, err := s.thru.ReadBytes(RecordSeparator)

	if errors.Is(err, io.EOF) && len(out) == 0 {
		return 0, err
	}

	if err != nil && !errors.Is(err, io.EOF) {
		n = copy(p, out)
		return n, err
	}

	if out[len(out)-1] == RecordSeparator {
		n = copy(p, out[:len(out)-1])
		s.eor = true
		return n, EOR.F(io.EOF)
	}

	n = copy(p, out)
	return n, nil
}

func newRecordScan(r io.Reader) io.Reader {
	_, isReader := r.(*reader)
	_, isBufioReader := r.(*bufio.Reader)
	return &recordScan{
		r:             bufio.NewReader(r),
		thru:          bufio.NewReader(strings.NewReader("")),
		eor:           false,
		buf:           new(bytes.Buffer),
		isBufioReader: isReader || isBufioReader,
	}
}
