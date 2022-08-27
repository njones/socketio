package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PayloadV3, string, bool, bool, error) testFn
		testParamsOutFn func(*testing.T) (PayloadV3, string, bool, bool, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(output PayloadV3, input string, clientIsBinary bool, clientIsXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have PayloadV3
				var dec = NewPayloadDecoderV3(strings.NewReader(input))
				dec.hasBinarySupport = clientIsBinary
				dec.hasXHR2Support = clientIsXHR2
				var err = dec.Decode(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"Encode": func(input PayloadV3, output string, clientIsBinary bool, clientIsXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var enc = NewPayloadEncoderV3(have)
				enc.hasBinarySupport = clientIsBinary
				enc.hasXHR2Support = clientIsXHR2
				var err = enc.Encode(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"ReadPayload": func(output PayloadV3, input string, clientIsBinary bool, clientIsXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have PayloadV3
				var err = NewPayloadDecoderV3.SetBinary(clientIsBinary).SetXHR2(clientIsXHR2).From(strings.NewReader(input)).ReadPayload(&have)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have)
			}
		},
		"WritePayload": func(input PayloadV3, output string, clientIsBinary bool, clientIsXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV3.SetBinary(clientIsBinary).SetXHR2(clientIsXHR2).To(have).WritePayload(input)

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
		"WritePayload packet": func(input PayloadV3, output string, clientIsBinary bool, clientIsXHR2 bool, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				t.Parallel()

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV3.SetBinary(clientIsBinary).SetXHR2(clientIsXHR2).To(have).WritePayload(input.PayloadVal())

				assert.ErrorIs(t, err, xErr)
				assert.Equal(t, output, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Without Binary": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			clientBinary, clientXHR2 := false, false
			asString := `6:4hello2:4€`
			asPayload := PayloadV3{
				{PacketV2{Packet{T: MessagePacket, D: "hello"}}, false},
				{PacketV2{Packet{T: MessagePacket, D: "€"}}, false},
			}
			return asPayload, asString, clientBinary, clientXHR2, nil
		},
		"Without Binary and Supported": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			clientBinary, clientXHR2 := false, true
			asString := `6:4hello2:4€`
			asPayload := PayloadV3{
				{PacketV2{Packet{T: MessagePacket, D: "hello"}}, false},
				{PacketV2{Packet{T: MessagePacket, D: "€"}}, false},
			}
			return asPayload, asString, clientBinary, clientXHR2, nil
		},
		"With Binary and Supported": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			clientBinary, clientXHR2 := true, true
			asString := string([]byte{0x00, 0x04, 0xff, 0x34, 0xe2, 0x82, 0xac, 0x01, 0x04, 0xff, 0x01, 0x02, 0x03, 0x04})
			asPayload := PayloadV3{
				{PacketV2{Packet{T: MessagePacket, D: "€"}}, false},
				{PacketV2{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}}, true},
			}
			return asPayload, asString, clientBinary, clientXHR2, nil
		},
		"With Binary and Not Supported": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			clientBinary, clientXHR2 := true, false
			asString := `2:4€10:b4AQIDBA==`
			asPayload := PayloadV3{
				{PacketV2{Packet{T: MessagePacket, D: "€"}}, false},
				{PacketV2{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}}, true},
			}
			return asPayload, asString, clientBinary, clientXHR2, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
