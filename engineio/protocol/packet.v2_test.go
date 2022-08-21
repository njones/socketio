//go:build gc || (eio_pac_v2 && eio_pac_v3)

package protocol

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testingName = strings.NewReplacer(" ", "_")

func runTest(testNames ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			if testName == "" || testName == "*" {
				return
			}

			want := testingName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				return
			}
		}
		t.SkipNow()
	}
}

func TestReadPacketV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data string, want PacketV2, xerr error) testFn
		testParamsOutFn func(*testing.T) (data string, want PacketV2, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(data string, want PacketV2, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PacketV2
				var err = NewPacketDecoderV2(strings.NewReader(data)).Decode(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
		"ReadPacket": func(data string, want PacketV2, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var decoder _packetDecoderV2 = NewPacketDecoderV2

				var have PacketV2
				var err = decoder.From(strings.NewReader(data)).ReadPacket(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (string, PacketV2, error) {
			data := `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`
			want := PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)}}}
			return data, want, nil
		},
		"Close": func(*testing.T) (string, PacketV2, error) {
			data := `1`
			want := PacketV2{Packet{T: ClosePacket, D: nil}}
			return data, want, nil
		},
		"Ping": func(*testing.T) (string, PacketV2, error) {
			data := `2`
			want := PacketV2{Packet{T: PingPacket, D: nil}}
			return data, want, nil
		},
		"Pong with Text": func(*testing.T) (string, PacketV2, error) {
			data := `3probe`
			want := PacketV2{Packet{T: PongPacket, D: "probe"}}
			return data, want, nil
		},
		"Message": func(*testing.T) (string, PacketV2, error) {
			data := `4HelloWorld`
			want := PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}
			return data, want, nil
		},
		"Upgrade": func(*testing.T) (string, PacketV2, error) {
			data := `5`
			want := PacketV2{Packet{T: UpgradePacket, D: nil}}
			return data, want, nil
		},
		"NOOP": func(*testing.T) (string, PacketV2, error) {
			data := `6`
			want := PacketV2{Packet{T: NoopPacket, D: nil}}
			return data, want, nil
		},

		// extra
		"Open Err JSON": func(*testing.T) (string, PacketV2, error) {
			data := `0{"sid":"abc1`
			err := ErrHandshakeDecode.F("v2", io.ErrUnexpectedEOF)
			return data, PacketV2{Packet{D: new(HandshakeV2)}}, err
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePacketV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data PacketV2, want string, xerr error) testFn
		testParamsOutFn func(*testing.T) (data PacketV2, want string, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Encode": func(data PacketV2, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var err = NewPacketEncoderV2(have).Encode(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
		"WritePacket": func(data PacketV2, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var encoder _packetEncoderV2 = NewPacketEncoderV2

				var have = new(bytes.Buffer)
				var err = encoder.To(have).WritePacket(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (PacketV2, string, error) {
			want := `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`
			data := PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{SID: "abc123", PingTimeout: Duration(300 * time.Millisecond)}}}
			return data, want, nil
		},
		"Close": func(*testing.T) (PacketV2, string, error) {
			want := `1`
			data := PacketV2{Packet{T: ClosePacket, D: nil}}
			return data, want, nil
		},
		"Ping": func(*testing.T) (PacketV2, string, error) {
			want := `2`
			data := PacketV2{Packet{T: PingPacket, D: nil}}
			return data, want, nil
		},
		"Pong with Text": func(*testing.T) (PacketV2, string, error) {
			want := `3probe`
			data := PacketV2{Packet{T: PongPacket, D: "probe"}}
			return data, want, nil
		},
		"Message": func(*testing.T) (PacketV2, string, error) {
			want := `4HelloWorld`
			data := PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}
			return data, want, nil
		},
		"Upgrade": func(*testing.T) (PacketV2, string, error) {
			want := `5`
			data := PacketV2{Packet{T: UpgradePacket, D: nil}}
			return data, want, nil
		},
		"NOOP": func(*testing.T) (PacketV2, string, error) {
			want := `6`
			data := PacketV2{Packet{T: NoopPacket, D: nil}}
			return data, want, nil
		},

		// extra
		"Err PacketType": func(*testing.T) (PacketV2, string, error) {
			data := PacketV2{Packet{T: 200, D: nil}}
			err := ErrInvalidPacketType
			return data, "", err
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
