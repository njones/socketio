//go:build gc || eio_pac_v3

package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPacketV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PacketV3, string, bool, error) testFn
		testParamsOutFn func(*testing.T) (PacketV3, string, bool, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(output PacketV3, input string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = PacketV3{IsBinary: isBinary}
				var err = NewPacketDecoderV3(strings.NewReader(input)).Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Encode": func(input PacketV3, output string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPacketEncoderV3(have).Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"ReadPacket": func(input PacketV3, output string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var decoder _packetDecoderV3 = NewPacketDecoderV3

				var have = PacketV3{IsBinary: isBinary}
				var err = decoder.From(strings.NewReader(output)).ReadPacket(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, input, have)
			}
		},
		"WritePacket": func(input PacketV3, output string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var encoder _packetEncoderV3 = NewPacketEncoderV3

				var have = new(bytes.Buffer)
				var err = encoder.To(have).WritePacket(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `0{"sid":"abc123","upgrades":[],"pingTimeout":300,"pingInterval":5000}`
			asPacket := PacketV3{
				PacketV2: PacketV2{
					Packet: Packet{
						T: OpenPacket,
						D: &HandshakeV3{
							HandshakeV2:  HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)},
							PingInterval: Duration(5000 * time.Millisecond),
						},
					},
				},
				IsBinary: false,
			}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},
		"Close": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `1`
			asPacket := PacketV3{PacketV2{Packet{T: ClosePacket, D: nil}}, false}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},
		"Ping": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `2`
			asPacket := PacketV3{PacketV2{Packet{T: PingPacket, D: nil}}, false}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},
		"Pong with Text": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `3probe`
			asPacket := PacketV3{PacketV2{Packet{T: PongPacket, D: "probe"}}, false}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},
		"Message": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `4HelloWorld`
			asPacket := PacketV3{PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}, false}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},
		"Message with Binary": func(*testing.T) (PacketV3, string, bool, error) {
			asString := "4\x00\x01\x02\x03\x04\x05"
			asPacket := PacketV3{PacketV2{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5})}}, true}
			asBinary := true
			return asPacket, asString, asBinary, nil
		},
		"Upgrade": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `5`
			asPacket := PacketV3{PacketV2{Packet{T: UpgradePacket, D: nil}}, false}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},
		"NOOP": func(*testing.T) (PacketV3, string, bool, error) {
			asString := `6`
			asPacket := PacketV3{PacketV2{Packet{T: NoopPacket, D: nil}}, false}
			asBinary := false
			return asPacket, asString, asBinary, nil
		},

		// extra
		"Message with Binary #2": func(*testing.T) (PacketV3, string, bool, error) {
			asString := "4\x00\x01\x02\x03\x04\x05"
			asPacket := PacketV3{PacketV2{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5})}}, true}
			asBinary := true
			return asPacket, asString, asBinary, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
