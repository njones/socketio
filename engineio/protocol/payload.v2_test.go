package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PayloadV2, string, error) testFn
		testParamsOutFn func(*testing.T) (PayloadV2, string, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(output PayloadV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have PayloadV2
				var err = NewPayloadDecoderV2(strings.NewReader(input)).Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Encode": func(input PayloadV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV2(have).Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"ReadPayload": func(output PayloadV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var got Payload
				var pay = _payloadDecoderV2(NewPayloadDecoderV2)
				var err = pay.From(strings.NewReader(input)).ReadPayload(&got)

				var have = make(PayloadV2, len(got))
				for i, v := range got {
					have[i] = PacketV2{v}
				}

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"WritePayload": func(input PayloadV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var pay = _payloadEncoderV2(NewPayloadEncoderV2)
				var err = pay.To(have).WritePayload(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"WritePayload packet": func(input PayloadV2, output string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var pay = _payloadEncoderV2(NewPayloadEncoderV2)
				var err = pay.To(have).WritePayload(input.PayloadVal())

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"ReadPayload packet": func(output PayloadV2, input string, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have Payload
				var pay = _payloadDecoderV2(NewPayloadDecoderV2)
				var err = pay.From(strings.NewReader(input)).ReadPayload(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output.PayloadVal(), have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Payload": func(*testing.T) (PayloadV2, string, error) {
			asString := `41:0{"sid":"","upgrades":[],"pingTimeout":0}1:16:2probe6:3probe11:4HelloWorld1:51:6`
			asPacket := PayloadV2{
				PacketV2{Packet{T: OpenPacket, D: &HandshakeV2{Upgrades: []string{}}}},
				PacketV2{Packet{T: ClosePacket}},
				PacketV2{Packet{T: PingPacket, D: "probe"}},
				PacketV2{Packet{T: PongPacket, D: "probe"}},
				PacketV2{Packet{T: MessagePacket, D: "HelloWorld"}},
				PacketV2{Packet{T: UpgradePacket}},
				PacketV2{Packet{T: NoopPacket}},
			}
			return asPacket, asString, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
