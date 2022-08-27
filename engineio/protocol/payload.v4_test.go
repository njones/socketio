package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadV4(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PayloadV4, string, bool, error) testFn
		testParamsOutFn func(*testing.T) (PayloadV4, string, bool, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(output PayloadV4, input string, isXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have PayloadV4
				var dec = NewPayloadDecoderV4(strings.NewReader(input))
				dec.hasXHR2Support = isXHR2
				var err = dec.Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Encode": func(input PayloadV4, output string, isXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var enc = NewPayloadEncoderV4(have)
				var err = enc.Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"ReadPayload": func(output PayloadV4, input string, isXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have PayloadV4
				var err = NewPayloadDecoderV4.From(strings.NewReader(input)).ReadPayload(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"WritePayload": func(input PayloadV4, output string, isXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV4.To(have).WritePayload(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Without Binary": func(*testing.T) (PayloadV4, string, bool, error) {
			isXHR2 := false
			asString := "4hello\x1e4€"
			asPacket := PayloadV4{
				{PacketV3{PacketV2{Packet{T: MessagePacket, D: "hello"}}, false}},
				{PacketV3{PacketV2{Packet{T: MessagePacket, D: "€"}}, false}},
			}
			return asPacket, asString, isXHR2, nil
		},
		"With Binary": func(*testing.T) (PayloadV4, string, bool, error) {
			isXHR2 := false
			asString := "4€\x1ebAQIDBA=="
			asPacket := PayloadV4{
				{PacketV3{PacketV2{Packet{T: MessagePacket, D: "€"}}, false}},
				{PacketV3{PacketV2{Packet{T: BinaryPacket, D: bytes.NewBuffer([]byte{0x01, 0x02, 0x03, 0x04})}}, true}},
			}
			return asPacket, asString, isXHR2, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
