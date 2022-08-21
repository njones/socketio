package protocol

import (
	"io"
	"strconv"
	"strings"
)

func (rdr *reader) packetLen() (n int64) {
	if rdr.IsErr() {
		return 0
	}

	var data string
	data, rdr.err = rdr.Bufio().ReadString(':')
	if rdr.err != nil {
		rdr.SetErr(rdr.err)
		return 0
	}
	n, rdr.err = strconv.ParseInt(strings.TrimRight(data, ":"), 10, 64)
	if rdr.err != nil {
		rdr.SetErr(rdr.err)
		return 0
	}

	return n
}

func (rdr *reader) payload(n int64) io.Reader {
	return LimitRuneReader(rdr.Bufio(), n)
}
