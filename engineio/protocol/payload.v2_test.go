package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadPayloadV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data string, want PayloadV2, xerr error) testFn
		testParamsOutFn func(*testing.T) (data string, want PayloadV2, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(data string, want PayloadV2, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PayloadV2
				var err = NewPayloadDecoderV2(strings.NewReader(data)).Decode(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
		"ReadPayload": func(data string, want PayloadV2, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var got Payload
				var pay = _payloadDecoderV2(NewPayloadDecoderV2)
				var err = pay.From(strings.NewReader(data)).ReadPayload(&got)

				var have = make(PayloadV2, len(got))
				for i, v := range got {
					have[i] = PacketV2{v}
				}

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
		"ReadPayload packet": func(data string, want PayloadV2, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have Payload
				var pay = _payloadDecoderV2(NewPayloadDecoderV2)
				var err = pay.From(strings.NewReader(data)).ReadPayload(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want.PayloadVal(), have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Payload": func(*testing.T) (string, PayloadV2, error) {
			data := `41:0{"sid":"","upgrades":[],"pingTimeout":0}1:16:2probe6:3probe11:4HelloWorld1:51:6`
			want := PayloadV2{
				PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{Upgrades: []string{}}}},
				PacketV2{Packet{T: ClosePacket}},
				PacketV2{Packet{T: PingPacket, D: "probe"}},
				PacketV2{Packet{T: PongPacket, D: "probe"}},
				PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}},
				PacketV2{Packet{T: UpgradePacket}},
				PacketV2{Packet{T: NoopPacket}},
			}
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePayloadV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data PayloadV2, want string, xerr error) testFn
		testParamsOutFn func(*testing.T) (data PayloadV2, want string, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Encode": func(data PayloadV2, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV2(have).Encode(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
		"WritePayload": func(data PayloadV2, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var pay = _payloadEncoderV2(NewPayloadEncoderV2)
				var err = pay.To(have).WritePayload(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
		"WritePayload packet": func(data PayloadV2, want string, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var pay = _payloadEncoderV2(NewPayloadEncoderV2)
				var err = pay.To(have).WritePayload(data.PayloadVal())

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Payload": func(*testing.T) (PayloadV2, string, error) {
			want := `41:0{"sid":"","upgrades":[],"pingTimeout":0}1:16:2probe6:3probe11:4HelloWorld1:51:6`
			data := PayloadV2{
				PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{}}},
				PacketV2{Packet{T: ClosePacket}},
				PacketV2{Packet{T: PingPacket, D: "probe"}},
				PacketV2{Packet{T: PongPacket, D: "probe"}},
				PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}},
				PacketV2{Packet{T: UpgradePacket}},
				PacketV2{Packet{T: NoopPacket}},
			}
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
