package protocol

import "io"

type erw struct {
	r io.Reader
	w io.Writer
}

func (rw erw) Read(p []byte) (int, error)  { return rw.r.Read(p) }
func (rw erw) Write(p []byte) (int, error) { return rw.w.Write(p) }

func underlining(w io.Writer, r io.Reader) (io.Writer, io.Reader) { return erw{w: w}, erw{r: r} }
