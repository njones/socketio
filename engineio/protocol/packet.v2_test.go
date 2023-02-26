//go:build gc || (eio_pac_v2 && eio_pac_v3)

package protocol

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	itst "github.com/njones/socketio/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestPacketV2(t *testing.T) {
	var opts = []func(*testing.T) bool{}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PacketV2, string, error) testFn
		testParamsOutFn func(*testing.T) (PacketV2, string, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(output PacketV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have PacketV2
				var err = NewPacketDecoderV2(strings.NewReader(input)).Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Encode": func(input PacketV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPacketEncoderV2(have).Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"ReadPacket": func(output PacketV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have PacketV2
				var decoder _packetDecoderV2 = NewPacketDecoderV2
				var err = decoder.From(strings.NewReader(input)).ReadPacket(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"WritePacket": func(input PacketV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var encoder _packetEncoderV2 = NewPacketEncoderV2

				var have = new(bytes.Buffer)
				var err = encoder.To(have).WritePacket(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"Short Decode": func(output PacketV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have PacketV2
				var reader = shortReader{r: strings.NewReader(input), max: 5, ran: *rand.New(rand.NewSource(5))}
				var err = NewPacketDecoderV2(reader).Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Short Encode": func(input PacketV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var writer = shortWriter{w: have, max: 5, ran: *rand.New(rand.NewSource(5))}
				var err = NewPacketEncoderV2(writer).Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"Short ReadPacket": func(output PacketV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have PacketV2
				var reader = shortReader{r: strings.NewReader(input), max: 5, ran: *rand.New(rand.NewSource(10))}
				var decoder _packetDecoderV2 = NewPacketDecoderV2
				var err = decoder.From(reader).ReadPacket(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Short WritePacket": func(input PacketV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var writer = shortWriter{w: have, max: 5, ran: *rand.New(rand.NewSource(10))}
				var encoder _packetEncoderV2 = NewPacketEncoderV2
				var err = encoder.To(writer).WritePacket(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (PacketV2, string, error) {
			asString := `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`
			asPacket := PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)}}}
			return asPacket, asString, nil
		},
		"Close": func(*testing.T) (PacketV2, string, error) {
			asString := `1`
			asPacket := PacketV2{Packet{T: ClosePacket, D: nil}}
			return asPacket, asString, nil
		},
		"Ping": func(*testing.T) (PacketV2, string, error) {
			asString := `2`
			asPacket := PacketV2{Packet{T: PingPacket, D: nil}}
			return asPacket, asString, nil
		},
		"Pong with Text": func(*testing.T) (PacketV2, string, error) {
			asString := `3probe`
			asPacket := PacketV2{Packet{T: PongPacket, D: "probe"}}
			return asPacket, asString, nil
		},
		"Message": func(*testing.T) (PacketV2, string, error) {
			asString := `4HelloWorld`
			asPacket := PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}
			return asPacket, asString, nil
		},
		"Upgrade": func(*testing.T) (PacketV2, string, error) {
			asString := `5`
			asPacket := PacketV2{Packet{T: UpgradePacket, D: nil}}
			return asPacket, asString, nil
		},
		"NOOP": func(*testing.T) (PacketV2, string, error) {
			asString := `6`
			asPacket := PacketV2{Packet{T: NoopPacket, D: nil}}
			return asPacket, asString, nil
		},

		// extra
		"Open Err JSON": func(*testing.T) (PacketV2, string, error) {
			opts = append(opts, itst.DoNotTest(
				"Encode",
				"WritePacket",
				"Short_Encode",
				"Short_WritePacket",
			))

			asString := `0{"sid":"abc1`
			err := ErrDecodeHandshakeFailed.F(io.ErrUnexpectedEOF).KV(ver, "v2")
			return PacketV2{}, asString, err
		},
		"Err PacketType": func(*testing.T) (PacketV2, string, error) {
			opts = append(opts, itst.DoNotTest(
				"Decode",
				"ReadPacket",
				"Short_Decode",
				"Short_ReadPacket",
			))

			asPacket := PacketV2{Packet{T: 200, D: nil}}
			err := ErrUnexpectedPacketType
			return asPacket, "", err
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
