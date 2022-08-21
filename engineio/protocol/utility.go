package protocol

import (
	"io"
	"unicode/utf8"
)

type limitRuneReader struct {
	r io.Reader
	n int64
}

func (lr *limitRuneReader) Read(p []byte) (n int, err error) {
	if lr.n <= 0 {
		return 0, io.EOF
	}

	var s int
	n = int(lr.n)
	if int64(len(p)) < lr.n {
		n = len(p)
	}
	_, err = lr.r.Read(p[s:n]) // 10
	for !utf8.Valid(p[0:n]) && n <= len(p) {
		s = n
		n++
		_, err = lr.r.Read(p[s:n])
	}

	lr.n -= int64(utf8.RuneCount(p[0:n]))
	return
}

func LimitRuneReader(r io.Reader, n int64) io.Reader { return &limitRuneReader{r: r, n: n} }
