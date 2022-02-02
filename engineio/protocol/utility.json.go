package protocol

import (
	"io"
	"strconv"
	"time"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	i, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}
	*d = Duration(time.Duration(i) * time.Millisecond)
	return err
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	c := strconv.Itoa(int(time.Duration(d) / time.Millisecond))
	return []byte(c), nil
}

// This is because of https://golang.org/src/encoding/json/stream.go?s=4272:4319#L173
type stripLastNewlineWriter struct{ w io.Writer }

func (snw *stripLastNewlineWriter) Write(p []byte) (n int, err error) {
	if n = len(p) - 1; p[n] != '\n' { // assumes this is not in the middle of a read...
		n++
	}
	return snw.w.Write(p[:n])
}
