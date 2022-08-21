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

func TestReadPacketV4(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data string, want PacketV4, xerr error) testFn
		testParamsOutFn func(*testing.T) (data string, want PacketV4, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(data string, want PacketV4, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PacketV4
				var err = NewPacketDecoderV4(strings.NewReader(data)).Decode(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
		"ReadPacket": func(data string, want PacketV4, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var decoder _packetDecoderV4 = NewPacketDecoderV4

				var have PacketV4
				var err = decoder.From(strings.NewReader(data)).ReadPacket(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (string, PacketV4, error) {
			data := `0{"sid":"abc123","upgrades":[],"pingTimeout":300,"pingInterval":5000}`
			want := PacketV4{
				PacketV3: PacketV3{
					Packet: Packet{
						T: OpenPacket,
						D: HandshakeV3{
							HandshakeV2: HandshakeV2{
								SID:         "abc123",
								Upgrades:    []string{},
								PingTimeout: Duration(300 * time.Millisecond),
							},
							PingInterval: Duration(5000 * time.Millisecond),
						},
					},
				},
			}
			return data, want, nil
		},
		"Close": func(*testing.T) (string, PacketV4, error) {
			data := `1`
			want := PacketV4{PacketV3: PacketV3{Packet{T: ClosePacket, D: nil}, false}}
			return data, want, nil
		},
		"Ping": func(*testing.T) (string, PacketV4, error) {
			data := `2`
			want := PacketV4{PacketV3: PacketV3{Packet{T: PingPacket, D: nil}, false}}
			return data, want, nil
		},
		"Pong with Text": func(*testing.T) (string, PacketV4, error) {
			data := `3probe`
			want := PacketV4{PacketV3: PacketV3{Packet{T: PongPacket, D: "probe"}, false}}
			return data, want, nil
		},
		"Message": func(*testing.T) (string, PacketV4, error) {
			data := `4HelloWorld`
			want := PacketV4{PacketV3: PacketV3{Packet{T: MessagePacket, D: "HelloWorld"}, false}}
			return data, want, nil
		},
		"Message with Binary": func(*testing.T) (string, PacketV4, error) {
			data := "4\x00\x01\x02\x03\x04\x05"
			want := PacketV4{PacketV3: PacketV3{Packet{T: MessagePacket, D: "\x00\x01\x02\x03\x04\x05"}, false}}
			return data, want, nil
		},
		"Upgrade": func(*testing.T) (string, PacketV4, error) {
			data := `5`
			want := PacketV4{PacketV3: PacketV3{Packet{T: UpgradePacket, D: nil}, false}}
			return data, want, nil
		},
		"NOOP": func(*testing.T) (string, PacketV4, error) {
			data := `6`
			want := PacketV4{PacketV3: PacketV3{Packet{T: NoopPacket, D: nil}, false}}
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePacketV4(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data PacketV4, want string, xerr error) testFn
		testParamsOutFn func(*testing.T) (data PacketV4, want string, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Encode": func(data PacketV4, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var err = NewPacketEncoderV4(have).Encode(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}

		},
		"WritePacket": func(data PacketV4, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var encoder _packetEncoderV4 = NewPacketEncoderV4

				var have = new(bytes.Buffer)
				var err = encoder.To(have).WritePacket(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (PacketV4, string, error) {
			want := `0{"sid":"abc123","upgrades":[],"pingTimeout":300,"pingInterval":5000}`
			data := PacketV4{
				PacketV3: PacketV3{
					Packet: Packet{
						T: OpenPacket,
						D: &HandshakeV3{
							HandshakeV2: HandshakeV2{
								SID:         "abc123",
								Upgrades:    []string{},
								PingTimeout: Duration(300 * time.Millisecond),
							},
							PingInterval: Duration(5000 * time.Millisecond),
						},
					},
				},
			}
			return data, want, nil
		},
		"Close": func(*testing.T) (PacketV4, string, error) {
			want := `1`
			data := PacketV4{PacketV3: PacketV3{Packet{T: ClosePacket, D: nil}, false}}
			return data, want, nil
		},
		"Ping": func(*testing.T) (PacketV4, string, error) {
			want := `2`
			data := PacketV4{PacketV3: PacketV3{Packet{T: PingPacket, D: nil}, false}}
			return data, want, nil
		},
		"Pong with Text": func(*testing.T) (PacketV4, string, error) {
			want := `3probe`
			data := PacketV4{PacketV3: PacketV3{Packet{T: PongPacket, D: "probe"}, false}}
			return data, want, nil
		},
		"Message": func(*testing.T) (PacketV4, string, error) {
			want := `4HelloWorld`
			data := PacketV4{PacketV3: PacketV3{Packet{T: MessagePacket, D: "HelloWorld"}, false}}
			return data, want, nil
		},
		"Message with Binary": func(*testing.T) (PacketV4, string, error) {
			want := "4\x00\x01\x02\x03\x04\x05"
			data := PacketV4{PacketV3: PacketV3{Packet{T: MessagePacket, D: "\x00\x01\x02\x03\x04\x05"}, false}}
			return data, want, nil
		},
		"Upgrade": func(*testing.T) (PacketV4, string, error) {
			want := `5`
			data := PacketV4{PacketV3: PacketV3{Packet{T: UpgradePacket, D: nil}, false}}
			return data, want, nil
		},
		"NOOP": func(*testing.T) (PacketV4, string, error) {
			want := `6`
			data := PacketV4{PacketV3: PacketV3{Packet{T: NoopPacket, D: nil}, false}}
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
