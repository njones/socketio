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

func TestReadPacketV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data string, isBin bool, want PacketV3, xerr error) testFn
		testParamsOutFn func(*testing.T) (data string, isBin bool, want PacketV3, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(data string, isBin bool, want PacketV3, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = PacketV3{IsBinary: isBin}
				var err = NewPacketDecoderV3(strings.NewReader(data)).Decode(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
		"ReadPacket": func(data string, isBin bool, want PacketV3, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var decoder _packetDecoderV3 = NewPacketDecoderV3

				var have = PacketV3{IsBinary: isBin}
				var err = decoder.From(strings.NewReader(data)).ReadPacket(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `0{"sid":"abc123","upgrades":[],"pingTimeout":300,"pingInterval":5000}`
			want := PacketV3{
				Packet: Packet{
					T: OpenPacket,
					D: HandshakeV3{
						HandshakeV2:  HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)},
						PingInterval: Duration(5000 * time.Millisecond),
					},
				},
				IsBinary: false,
			}
			return data, isBin, want, nil
		},
		"Close": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `1`
			want := PacketV3{Packet{T: ClosePacket, D: nil}, false}
			return data, isBin, want, nil
		},
		"Ping": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `2`
			want := PacketV3{Packet{T: PingPacket, D: nil}, false}
			return data, isBin, want, nil
		},
		"Pong with Text": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `3probe`
			want := PacketV3{Packet{T: PongPacket, D: "probe"}, false}
			return data, isBin, want, nil
		},
		"Message": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `4HelloWorld`
			want := PacketV3{Packet{T: MessagePacket, D: "HelloWorld"}, false}
			return data, isBin, want, nil
		},
		"Message with Binary": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := "4\x00\x01\x02\x03\x04\x05"
			want := PacketV3{Packet{T: MessagePacket, D: "\x00\x01\x02\x03\x04\x05"}, false}
			return data, isBin, want, nil
		},
		"Upgrade": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `5`
			want := PacketV3{Packet{T: UpgradePacket, D: nil}, false}
			return data, isBin, want, nil
		},
		"NOOP": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin bool
			data := `6`
			want := PacketV3{Packet{T: NoopPacket, D: nil}, false}
			return data, isBin, want, nil
		},

		// extra
		"Message with Binary #2": func(*testing.T) (string, bool, PacketV3, error) {
			var isBin = true
			data := "4\x00\x01\x02\x03\x04\x05"
			want := PacketV3{Packet{T: MessagePacket, D: bytes.NewReader([]byte("\x00\x01\x02\x03\x04\x05"))}, true}
			return data, isBin, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePacketV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data PacketV3, want string, xerr error) testFn
		testParamsOutFn func(*testing.T) (data PacketV3, want string, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Encode": func(data PacketV3, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var err = NewPacketEncoderV3(have).Encode(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
		"WritePacket": func(data PacketV3, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var encoder _packetEncoderV3 = NewPacketEncoderV3

				var have = new(bytes.Buffer)
				var err = encoder.To(have).WritePacket(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Open": func(*testing.T) (PacketV3, string, error) {
			want := `0{"sid":"abc123","upgrades":[],"pingTimeout":300,"pingInterval":5000}`
			data := PacketV3{
				Packet: Packet{
					T: OpenPacket,
					D: &HandshakeV3{
						HandshakeV2:  HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)},
						PingInterval: Duration(5000 * time.Millisecond),
					},
				},
				IsBinary: false,
			}
			return data, want, nil
		},
		"Close": func(*testing.T) (PacketV3, string, error) {
			want := `1`
			data := PacketV3{Packet{T: ClosePacket, D: nil}, false}
			return data, want, nil
		},
		"Ping": func(*testing.T) (PacketV3, string, error) {
			data := PacketV3{Packet{T: PingPacket, D: nil}, false}
			want := `2`
			return data, want, nil
		},
		"Pong with Text": func(*testing.T) (PacketV3, string, error) {
			want := `3probe`
			data := PacketV3{Packet{T: PongPacket, D: "probe"}, false}
			return data, want, nil
		},
		"Message": func(*testing.T) (PacketV3, string, error) {
			want := `4HelloWorld`
			data := PacketV3{Packet{T: MessagePacket, D: "HelloWorld"}, false}
			return data, want, nil
		},
		"Message with Binary": func(*testing.T) (PacketV3, string, error) {
			want := "4\x00\x01\x02\x03\x04\x05"
			data := PacketV3{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5})}, false}
			return data, want, nil
		},
		"Upgrade": func(*testing.T) (PacketV3, string, error) {
			want := `5`
			data := PacketV3{Packet{T: UpgradePacket, D: nil}, false}
			return data, want, nil
		},
		"NOOP": func(*testing.T) (PacketV3, string, error) {
			want := `6`
			data := PacketV3{Packet{T: NoopPacket, D: nil}, false}
			return data, want, nil
		},

		// extra
		"Message with Binary #2": func(*testing.T) (PacketV3, string, error) {
			want := "4\x00\x01\x02\x03\x04\x05"
			data := PacketV3{Packet{T: MessagePacket, D: "\x00\x01\x02\x03\x04\x05"}, false}
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
