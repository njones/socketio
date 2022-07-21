package protocol

import (
	"bufio"
	"bytes"
	"errors"
	"io"
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
