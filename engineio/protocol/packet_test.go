package protocol

import (
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	itst "github.com/njones/socketio/internal/test"
	"github.com/stretchr/testify/assert"
)

var runTest, skipTest = itst.RunTest, itst.SkipTest
var _, _ = runTest, skipTest

type shortReader struct {
	max int
	ran rand.Rand
	r   io.Reader
}

func (sr shortReader) Read(p []byte) (n int, err error) {
	var x = 100
	for x > 0 && err == nil {
		i := n + sr.ran.Intn(sr.max) + 1
		if len(p) < i {
			i = len(p)
		}
		x, err = sr.r.Read(p[n:i])
		n += x
	}
	return n, err
}

type shortWriter struct {
	max int
	ran rand.Rand
	w   io.Writer
}

func (sw shortWriter) Write(p []byte) (n int, err error) {
	var x, j = len(p), 0
	for n < x && err == nil {
		i := n + sw.ran.Intn(sw.max) + 1
		if x < i {
			i = x
		}
		j, err = sw.w.Write(p[n:i])
		n += j
	}
	return n, err
}

func TestPacketLength(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data Packet, want int) testFn
		testParamsOutFn func(*testing.T) (data Packet, want int)
	)

	runWithOptions := map[string]testParamsInFn{
		"Basic": func(data Packet, want int) testFn {
			return func(*testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				length := data.Len()
				assert.Equal(t, want, length)
			}
		},
	}

	var values = func(a Packet, b int) func(*testing.T) (Packet, int) {
		return func(*testing.T) (Packet, int) { return a, b }
	}

	tests := map[string]testParamsOutFn{
		"Open":              values(Packet{T: OpenPacket, D: new(HandshakeV2)}, 41),
		"Close":             values(Packet{T: ClosePacket}, 1),
		"Message with Text": values(Packet{T: MessagePacket, D: "HelloWorld"}, 11),
		"Ping with text":    values(Packet{T: PingPacket, D: "probe"}, 6),
		"Open with Handshake v2": values(Packet{T: OpenPacket, D: &HandshakeV2{
			SID:         "The ID here",
			Upgrades:    []string{},
			PingTimeout: Duration(5000 * time.Millisecond),
		}}, len(`0{"sid":"The ID here","upgrades":[],"pingTimeout":5000}`)),
		"Open with Handshake v3": values(Packet{T: OpenPacket, D: &HandshakeV3{
			HandshakeV2: &HandshakeV2{
				SID:         "The ID here",
				Upgrades:    []string{},
				PingTimeout: Duration(5000 * time.Millisecond),
			},
			PingInterval: Duration(5000 * time.Millisecond),
		}}, len(`0{"sid":"The ID here","upgrades":[],"pingTimeout":5000,"pingInterval":5000}`)),
	}

	for name, testParams := range tests {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
