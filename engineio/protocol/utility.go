package protocol

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// CopyRuneN copies n runes (or until an error) from src to dst. It returns the number
// of runes copied and the earliest error encountered while copying. On return, written
// == n, if and only if err == nil.
func CopyRuneN(dst io.Writer, src io.Reader, n int64) (written int64, err error) {
	// this is the first implentation... so it may not be really efficient...
	// but as of now it works and passes tests...
	if n == 0 {
		return n, nil
	}

	var (
		overflow  [3]byte
		prevx, x  int
		prev, cnt int64
	)

ReadMore:
	adv := (n - cnt) + int64(x)
	buf := bufio.NewReader(io.LimitReader(io.MultiReader(bytes.NewReader(overflow[:x]), src), adv))
	prevx, x = x, 0 // clear for the next go round...
	for ; cnt < n; cnt++ {
		r, size, err := buf.ReadRune()
		if r == utf8.RuneError && size == 1 {
			buf.UnreadRune()
			x, err = buf.Read(overflow[:])
			if err != nil {
				return cnt, err
			}
			if x == prevx {
				return cnt, ErrInvalidRune
			}
			goto ReadMore
		}
		prevx = 0 // clear because it's outside of the RuneError
		if err != nil && errors.Is(err, io.EOF) {
			if cnt > 0 && prev == cnt { // if no change then we're done
				return cnt, err
			}
			prev = cnt
			goto ReadMore
		}
		if err != nil {
			return cnt, err
		}
		_, err = dst.Write([]byte(string(r)))
		if err != nil {
			return cnt, err
		}
	}
	return cnt, err
}

// head formats the header for v3 payload encoding
func head(length uint64, isBinary bool) []byte {
	var x uint64
	if isBinary {
		x = 1
	}

	blen := Ltob(length - x)
	header := make([]byte, len(blen)+2)
	copy(header[1:], blen)
	header[len(header)-1] = 0xFF

	header[0] = byte(x)
	return header
}

// Ltob is len to bytes conversion. This converts a number to a []byte slice
// 455 => []byte{4,5,5}
func Ltob(u uint64) []byte {
	s := strconv.FormatUint(u, 10)
	p := make([]byte, len(s))
	for i, k := range []byte(s) {
		p[i] = k & 0x0F
	}
	return p
}

// Btol is bytes to len conversion. This []byte slice to a number
// []byte{4,5,5} => 455
func Btol(p []byte) uint64 {
	var s = new(strings.Builder)
	for _, k := range p {
		s.WriteString(fmt.Sprintf("%d", k))
	}
	u, err := strconv.ParseUint(s.String(), 10, 64)
	if err != nil {
		panic(err)
	}
	return u
}
