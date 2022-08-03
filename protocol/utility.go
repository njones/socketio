package protocol

import (
	"io"
	"math"
)

// erw wraps an io.Reader and io.Writer so that it uses the .Read and .Write
// methods, and not the .ReadFrom or .WriteTo methods, which would create a
// cycle if used within one of the Packets ReadFrom/WriteTo methods.
type erw struct {
	r io.Reader
	w io.Writer
}

// Read will force the use of the Read method and not ReadFrom
func (rw erw) Read(p []byte) (int, error) { return rw.r.Read(p) }

// Write will force the use of the Write method and not WriteTo
func (rw erw) Write(p []byte) (int, error) { return rw.w.Write(p) }

// underlining returns erw wrapped Read and Write methods for func(dst io.Writer, src io.Reader) signatures
func underlining(w io.Writer, r io.Reader) (io.Writer, io.Reader) { return erw{w: w}, erw{r: r} }

func intLen(v interface{}) int {
	var i float64
	switch x := v.(type) {
	case int:
		i = float64(x)
	case int8:
		i = float64(x)
	case int16:
		i = float64(x)
	case int32:
		i = float64(x)
	case int64:
		i = float64(x)
	case uint8:
		i = float64(x)
	case uint16:
		i = float64(x)
	case uint32:
		i = float64(x)
	case uint64:
		i = float64(x)
	case float32:
		i = float64(x)
	case float64:
		i = x
	}
	return int(math.Floor(math.Log10(i))) + 1
}
