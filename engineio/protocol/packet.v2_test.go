//go:build gc || (eio_pac_v2 && eio_pac_v3)

package protocol

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testoption func(*testing.T)

func runTest(testnames ...string) testoption {
	return func(t *testing.T) {
		t.Helper()
		names := strings.SplitN(t.Name(), "/", 2)
		for _, testname := range testnames {
			if names[len(names)-1] == strings.ReplaceAll(testname, " ", "_") {
				return
			}
		}
		t.SkipNow()
	}
}

func TestReadPacketV2(t *testing.T) {
	var opts []testoption

	runWithOptions := map[string]func(opts ...testoption) func(string, PacketV2, error) func(*testing.T){
		".Decode": func(opts ...testoption) func(string, PacketV2, error) func(*testing.T) {
			return func(data string, want PacketV2, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have PacketV2
					var err = NewPacketDecoderV2(strings.NewReader(data)).Decode(&have)

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have)
				}
			}
		},
		".ReadPacket": func(opts ...testoption) func(string, PacketV2, error) func(*testing.T) {
			return func(data string, want PacketV2, xerr error) func(*testing.T) {
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
			}
		},
	}

	spec := map[string]func() (string, PacketV2, error){
		"Open": func() (string, PacketV2, error) {
			data := `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`
			want := PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)}}}
			return data, want, nil
		},
		"Close": func() (string, PacketV2, error) {
			data := `1`
			want := PacketV2{Packet{T: ClosePacket, D: nil}}
			return data, want, nil
		},
		"Ping": func() (string, PacketV2, error) {
			data := `2`
			want := PacketV2{Packet{T: PingPacket, D: nil}}
			return data, want, nil
		},
		"Pong with Text": func() (string, PacketV2, error) {
			data := `3probe`
			want := PacketV2{Packet{T: PongPacket, D: "probe"}}
			return data, want, nil
		},
		"Message": func() (string, PacketV2, error) {
			data := `4HelloWorld`
			want := PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}
			return data, want, nil
		},
		"Upgrade": func() (string, PacketV2, error) {
			data := `5`
			want := PacketV2{Packet{T: UpgradePacket, D: nil}}
			return data, want, nil
		},
		"NOOP": func() (string, PacketV2, error) {
			data := `6`
			want := PacketV2{Packet{T: NoopPacket, D: nil}}
			return data, want, nil
		},
	}

	errs := map[string]func() (string, PacketV2, error){
		"Open Err JSON": func() (string, PacketV2, error) {
			data := `0{"sid":"abc1`
			err := ErrHandshakeDecode.F("v2", io.ErrUnexpectedEOF)
			return data, PacketV2{Packet{D: new(HandshakeV2)}}, err
		},
	}

	for name, testing := range spec {
		for suffix, runWithOption := range runWithOptions {
			t.Run(name+suffix, runWithOption(opts...)(testing()))
		}
	}

	for name, testing := range errs {
		for suffix, runWithOption := range runWithOptions {
			t.Run(name+suffix, runWithOption(opts...)(testing()))
		}
	}
}

func TestWritePacketV2(t *testing.T) {
	var opts []testoption

	runWithOptions := map[string]func(opts ...testoption) func(PacketV2, string, error) func(*testing.T){
		".Encode": func(opts ...testoption) func(PacketV2, string, error) func(*testing.T) {
			return func(data PacketV2, want string, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have = new(bytes.Buffer)
					var err = NewPacketEncoderV2(have).Encode(data)

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have.String())
				}
			}
		},
		".WritePacket": func(opts ...testoption) func(PacketV2, string, error) func(*testing.T) {
			return func(data PacketV2, want string, xerr error) func(*testing.T) {
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
			}
		},
	}

	spec := map[string]func() (PacketV2, string, error){
		"Open": func() (PacketV2, string, error) {
			want := `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`
			data := PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{SID: "abc123", PingTimeout: Duration(300 * time.Millisecond)}}}
			return data, want, nil
		},
		"Close": func() (PacketV2, string, error) {
			want := `1`
			data := PacketV2{Packet{T: ClosePacket, D: nil}}
			return data, want, nil
		},
		"Ping": func() (PacketV2, string, error) {
			want := `2`
			data := PacketV2{Packet{T: PingPacket, D: nil}}
			return data, want, nil
		},
		"Pong with Text": func() (PacketV2, string, error) {
			want := `3probe`
			data := PacketV2{Packet{T: PongPacket, D: "probe"}}
			return data, want, nil
		},
		"Message": func() (PacketV2, string, error) {
			want := `4HelloWorld`
			data := PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}}
			return data, want, nil
		},
		"Upgrade": func() (PacketV2, string, error) {
			want := `5`
			data := PacketV2{Packet{T: UpgradePacket, D: nil}}
			return data, want, nil
		},
		"NOOP": func() (PacketV2, string, error) {
			want := `6`
			data := PacketV2{Packet{T: NoopPacket, D: nil}}
			return data, want, nil
		},
	}

	errs := map[string]func() (PacketV2, string, error){
		"Err PacketType": func() (PacketV2, string, error) {
			data := PacketV2{Packet{T: 200, D: nil}}
			err := ErrInvalidPacketType
			return data, "", err
		},
	}

	for name, testing := range spec {
		for suffix, runWithOption := range runWithOptions {
			t.Run(name+suffix, runWithOption(opts...)(testing()))
		}
	}

	for name, testing := range errs {
		for suffix, runWithOption := range runWithOptions {
			t.Run(name+suffix, runWithOption(opts...)(testing()))
		}
	}
}
