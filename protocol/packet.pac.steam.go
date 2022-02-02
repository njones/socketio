package protocol

import (
	"io"
	"strconv"
)

type binaryStreamOut struct {
	idx int
	rdr []io.Reader
}

func (x *binaryStreamOut) Next() bool { return x.idx < len(x.rdr) }
func (x *binaryStreamOut) WriteTo(w io.Writer) (n int64, err error) {
	n, err = io.Copy(w, x.rdr[x.idx])
	x.idx++
	return
}

type binaryStreamIn []func(io.Reader) error

func (x binaryStreamIn) Read(p []byte) (n int, err error) {
	if len(x) == 0 {
		return
	}

	numStr := strconv.Itoa(len(x)) + "-"
	n = copy(p, numStr)

	if n < len(numStr) {
		return n, PacketError{str: "buffer binary stream for read", buffer: []byte(numStr)[n:], errs: []error{ErrShortRead}}
	}

	return n, nil
}

func (x *binaryStreamIn) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	var data []byte
	if x != nil && len(*x) > 0 { // this means we have a short write, so pick up where we left off...
		data = []byte(strconv.Itoa(len(*x)))
	}

	for i, val := range p {
		n++
		switch val {
		case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
			data = append(data, val)
			continue
		case '-':
			k, err := strconv.ParseInt(string(data), 10, 64)
			*x = make([]func(io.Reader) error, k)
			if err != nil {
				err = ErrBadParse.F("incoming binary stream", err)
			}
			return n, err
		}
		if i == 0 {
			return 0, nil
		}
	}

	num, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		err = ErrBadParse.F("incoming binary stream", err)
		return n, err
	}

	*x = make([]func(io.Reader) error, num)

	return n, ErrUnexpectedAttachmentEnd
}
