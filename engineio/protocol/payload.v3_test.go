package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadPayloadV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data string, want PayloadV3, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn
		testParamsOutFn func(*testing.T) (data string, want PayloadV3, hasBinaryClient bool, hasXHR2Client bool, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Decode": func(data string, want PayloadV3, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PayloadV3
				var dec = NewPayloadDecoderV3(strings.NewReader(data))
				dec.hasBinarySupport = hasBinaryClient
				dec.hasXHR2Support = hasXHR2Client
				var err = dec.Decode(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
		"ReadPayload": func(data string, want PayloadV3, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PayloadV3
				var err = NewPayloadDecoderV3.SetBinary(hasBinaryClient).SetXHR2(hasXHR2Client).From(strings.NewReader(data)).ReadPayload(&have)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Without Binary": func(*testing.T) (string, PayloadV3, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := false, false
			data := `6:4hello2:4€`
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, false},
				{Packet{T: MessagePacket, D: "€"}, false},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
		"Without Binary and Supported": func(*testing.T) (string, PayloadV3, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := false, true
			data := `6:4hello2:4€`
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, false},
				{Packet{T: MessagePacket, D: "€"}, false},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
		"With Binary and Supported": func(*testing.T) (string, PayloadV3, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := true, true
			data := string([]byte{0x00, 0x04, 0xff, 0x34, 0xe2, 0x82, 0xac, 0x01, 0x04, 0xff, 0x01, 0x02, 0x03, 0x04})
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, false},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, true},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
		"With Binary and Not Supported": func(*testing.T) (string, PayloadV3, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := true, false
			data := `2:4€10:b4AQIDBA==`
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, false},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, true},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePayloadV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(data PayloadV3, want string, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn
		testParamsOutFn func(*testing.T) (data PayloadV3, want string, hasBinaryClient bool, hasXHR2Client bool, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Encode": func(data PayloadV3, want string, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var enc = NewPayloadEncoderV3(have)
				enc.hasBinarySupport = hasBinaryClient
				enc.hasXHR2Support = hasXHR2Client
				var err = enc.Encode(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
		"WritePayload": func(data PayloadV3, want string, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV3.SetBinary(hasBinaryClient).SetXHR2(hasXHR2Client).To(have).WritePayload(data)

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
		"WritePayload packet": func(data PayloadV3, want string, hasBinaryClient bool, hasXHR2Client bool, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				var err = NewPayloadEncoderV3.SetBinary(hasBinaryClient).SetXHR2(hasXHR2Client).To(have).WritePayload(data.PayloadVal())

				assert.ErrorIs(t, err, xerr)
				assert.Equal(t, want, have.String())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Without Binary": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := false, false
			want := `6:4hello2:4€`
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, false},
				{Packet{T: MessagePacket, D: "€"}, false},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
		"Without Binary and Supported": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := false, true
			want := `6:4hello2:4€`
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, false},
				{Packet{T: MessagePacket, D: "€"}, false},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
		"With Binary and Supported": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := true, true
			want := string([]byte{0x00, 0x04, 0xff, 0x34, 0xe2, 0x82, 0xac, 0x01, 0x04, 0xff, 0x01, 0x02, 0x03, 0x04})
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, false},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, true},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
		"With Binary and Not Supported": func(*testing.T) (PayloadV3, string, bool, bool, error) {
			hasBinaryClient, hasXHR2Client := true, false
			want := `2:4€10:b4AQIDBA==`
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, false},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, true},
			}
			return data, want, hasBinaryClient, hasXHR2Client, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
