package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadPayloadV3(t *testing.T) {
	var opts []testoption

	runWithOptions := map[string]func(opts ...testoption) func(string, PayloadV3, bool, error) func(*testing.T){
		".Decode": func(opts ...testoption) func(string, PayloadV3, bool, error) func(*testing.T) {
			return func(data string, want PayloadV3, isXHR2 bool, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have PayloadV3
					var dec = NewPayloadDecoderV3(strings.NewReader(data))
					dec.IsXHR2 = isXHR2
					var err = dec.Decode(&have)

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have)
				}
			}
		},
		".ReadPayload": func(opts ...testoption) func(string, PayloadV3, bool, error) func(*testing.T) {
			return func(data string, want PayloadV3, isXHR2 bool, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have PayloadV3
					var err = NewPayloadDecoderV3.SetXHR2(isXHR2).From(strings.NewReader(data)).ReadPayload(&have)

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have)
				}
			}
		},
	}

	spec := map[string]func() (string, PayloadV3, bool, error){
		"Without Binary": func() (string, PayloadV3, bool, error) {
			isBinary, isXHR2 := false, false
			data := `6:4hello2:4€`
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, isBinary},
				{Packet{T: MessagePacket, D: "€"}, isBinary},
			}
			return data, want, isXHR2, nil
		},
		"Without Binary and Supported": func() (string, PayloadV3, bool, error) {
			isBinary, isXHR2 := false, true
			data := `6:4hello2:4€`
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, isBinary},
				{Packet{T: MessagePacket, D: "€"}, isBinary},
			}
			return data, want, isXHR2, nil
		},
		"With Binary and Supported": func() (string, PayloadV3, bool, error) {
			isBinary, isXHR2 := true, true
			data := string([]byte{0x00, 0x04, 0xff, 0x34, 0xe2, 0x82, 0xac, 0x01, 0x04, 0xff, 0x01, 0x02, 0x03, 0x04})
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, false},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, isBinary},
			}
			return data, want, isXHR2, nil
		},
		"With Binary and Not Supported": func() (string, PayloadV3, bool, error) {
			isBinary, isXHR2 := true, false
			data := `2:4€10:b4AQIDBA==`
			want := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, false},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, isBinary},
			}
			return data, want, isXHR2, nil
		},
	}

	for name, testing := range spec {
		for suffix, runWithOption := range runWithOptions {
			t.Run(name+suffix, runWithOption(opts...)(testing()))
		}
	}
}

func TestWritePayloadV3(t *testing.T) {
	var opts []testoption

	runWithOptions := map[string]func(opts ...testoption) func(PayloadV3, string, bool, error) func(*testing.T){
		".Encode": func(opts ...testoption) func(PayloadV3, string, bool, error) func(*testing.T) {
			return func(data PayloadV3, want string, isXHR2 bool, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have = new(bytes.Buffer)
					var enc = NewPayloadEncoderV3(have)
					enc.IsXHR2 = isXHR2
					var err = enc.Encode(data)

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have.String())
				}
			}
		},
		".WritePayload": func(opts ...testoption) func(PayloadV3, string, bool, error) func(*testing.T) {
			return func(data PayloadV3, want string, isXHR2 bool, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have = new(bytes.Buffer)
					var err = NewPayloadEncoderV3.SetXHR2(isXHR2).To(have).WritePayload(data)

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have.String())
				}
			}
		},
		".WritePayload packet": func(opts ...testoption) func(PayloadV3, string, bool, error) func(*testing.T) {
			return func(data PayloadV3, want string, isXHR2 bool, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, opt := range opts {
						opt(t)
					}

					var have = new(bytes.Buffer)
					var err = NewPayloadEncoderV3.SetXHR2(isXHR2).To(have).WritePayload(data.PayloadVal())

					assert.ErrorIs(t, err, xerr)
					assert.Equal(t, want, have.String())
				}
			}
		},
	}

	spec := map[string]func() (PayloadV3, string, bool, error){
		"Without Binary": func() (PayloadV3, string, bool, error) {
			isBinary, isXHR2 := false, false
			want := `6:4hello2:4€`
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, isBinary},
				{Packet{T: MessagePacket, D: "€"}, isBinary},
			}
			return data, want, isXHR2, nil
		},
		"Without Binary and Supported": func() (PayloadV3, string, bool, error) {
			isBinary, isXHR2 := false, true
			want := `6:4hello2:4€`
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "hello"}, isBinary},
				{Packet{T: MessagePacket, D: "€"}, isBinary},
			}
			return data, want, isXHR2, nil
		},
		"With Binary and Supported": func() (PayloadV3, string, bool, error) {
			isBinary, isXHR2 := true, true
			want := string([]byte{0x00, 0x04, 0xff, 0x34, 0xe2, 0x82, 0xac, 0x01, 0x04, 0xff, 0x01, 0x02, 0x03, 0x04})
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, isBinary},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, isBinary},
			}
			return data, want, isXHR2, nil
		},
		"With Binary and Not Supported": func() (PayloadV3, string, bool, error) {
			isBinary, isXHR2 := true, false
			want := `2:4€10:b4AQIDBA==`
			data := PayloadV3{
				{Packet{T: MessagePacket, D: "€"}, isBinary},
				{Packet{T: MessagePacket, D: bytes.NewReader([]byte{0x01, 0x02, 0x03, 0x04})}, isBinary},
			}
			return data, want, isXHR2, nil
		},
	}

	for name, testing := range spec {
		for suffix, runWithOption := range runWithOptions {
			t.Run(name+suffix, runWithOption(opts...)(testing()))
		}
	}
}
