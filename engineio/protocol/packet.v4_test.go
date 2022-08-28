//go:build gc || eio_pac_V4

package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPacketV4(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PacketV4, string, bool, error) testFn
		testParamsOutFn func(*testing.T) (PacketV4, string, bool, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(output PacketV4, input string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = PacketV4{PacketV3{IsBinary: isBinary}}
				var err = NewPacketDecoderV4(strings.NewReader(input)).Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Encode": func(input PacketV4, output string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPacketEncoderV4(have).Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}

		},
		"ReadPacket": func(output PacketV4, input string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = PacketV4{PacketV3{IsBinary: isBinary}}
				var decoder _packetDecoderV4 = NewPacketDecoderV4
				var err = decoder.From(strings.NewReader(input)).ReadPacket(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"WritePacket": func(input PacketV4, output string, isBinary bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var encoder _packetEncoderV4 = NewPacketEncoderV4
				var err = encoder.To(have).WritePacket(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `0{"sid":"abc123","upgrades":[],"pingTimeout":300,"pingInterval":5000,"maxPayload":10000}`
			isPacket := PacketV4{
				PacketV3: PacketV3{
					PacketV2: PacketV2{
						Packet: Packet{
							T: OpenPacket,
							D: &HandshakeV4{
								HandshakeV3: &HandshakeV3{
									HandshakeV2: &HandshakeV2{
										SID:         "abc123",
										Upgrades:    []string{},
										PingTimeout: Duration(300 * time.Millisecond),
									},
									PingInterval: Duration(5000 * time.Millisecond),
								},
								MaxPayload: 10000,
							},
						},
					},
				},
			}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
		"Close": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `1`
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: ClosePacket, D: nil}}, false}}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
		"Ping": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `2`
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: PingPacket, D: nil}}, false}}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
		"Pong with Text": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `3probe`
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: PongPacket, D: "probe"}}, false}}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
		"Message": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `4HelloWorld`
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}, false}}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
		"Message with Binary": func(*testing.T) (PacketV4, string, bool, error) {
			isString := "4\x00\x01\x02\x03\x04\x05"
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5})}}, true}}
			isBinary := true
			return isPacket, isString, isBinary, nil
		},
		"Upgrade": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `5`
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: UpgradePacket, D: nil}}, false}}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
		"NOOP": func(*testing.T) (PacketV4, string, bool, error) {
			isString := `6`
			isPacket := PacketV4{PacketV3: PacketV3{PacketV2{Packet{T: NoopPacket, D: nil}}, false}}
			isBinary := false
			return isPacket, isString, isBinary, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
