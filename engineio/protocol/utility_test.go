package protocol

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitRuneReader(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(io.Reader, int64, string, int64, error) testFn
		testParamsOutFn func(*testing.T) (io.Reader, int64, string, int64, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Basic": func(r io.Reader, limit int64, out string, chars int64, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				buf := new(bytes.Buffer)
				n, err := buf.ReadFrom(LimitRuneReader(r, limit))

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, out, buf.String())
				assert.Equal(t, chars, n)
			}
		},
	}

	tests := map[string]testParamsOutFn{
		"Standard": func(t *testing.T) (r io.Reader, limit int64, out string, chars int64, xerr error) {
			return strings.NewReader("0123456789"), 5, "01234", 5, nil
		},
		"Unicode": func(t *testing.T) (r io.Reader, limit int64, out string, chars int64, xerr error) {
			// int64(len([]byte("åß∂ƒ©"))) = 11
			return strings.NewReader("åß∂ƒ©˙∆˚¬…æ"), 5, "åß∂ƒ©", 11, nil
		},
		"mixed": func(t *testing.T) (r io.Reader, limit int64, out string, chars int64, xerr error) {
			// int64(len([]byte("åß∂ƒ©"))) = 11
			return strings.NewReader("å1ß2∂3ƒ4©5˙6∆7˚8¬9…0æ"), 5, "å1ß2∂", 9, nil
		},
	}

	for name, testParams := range tests {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func FuzzLimitRuneReader(f *testing.F) {
	f.Add("1234", int64(5))
	f.Add("åß∂ƒ©˙∆˚¬…æ", int64(3))
	f.Fuzz(func(t *testing.T, in string, n int64) {
		buf := new(bytes.Buffer)
		j, err := buf.ReadFrom(LimitRuneReader(strings.NewReader(in), n))
		assert.Equal(t, string([]rune(in)[:n]), buf.String())
		if err != nil {
			t.Errorf("bad: %q %d", buf.String(), j)
		}
	})
}
