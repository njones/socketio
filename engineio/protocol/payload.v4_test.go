package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadEncodeV4(t *testing.T) {
	type want struct {
		string
		error
	}
	var tests = []struct {
		name string
		data PayloadV4
		want want
	}{
		{
			name: "no binary",
			data: []PacketV4{
				{Packet: Packet{T: MessagePacket, D: "hello"}},
				{Packet: Packet{T: MessagePacket, D: "€"}},
			},
			want: want{error: nil, string: "4hello\x1e4€"},
		},
		{
			name: "with binary",
			data: []PacketV4{
				{Packet: Packet{T: MessagePacket, D: "€"}},
				{Packet: Packet{T: MessagePacket, D: []byte{1, 2, 3, 4}}},
			},
			want: want{error: nil, string: "4€\x1ebAQIDBA=="},
		},
	}

	buf := new(bytes.Buffer)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()

			err := NewPayloadEncoderV4(buf).Encode(test.data)

			assert.Equal(t, test.want.error, err)
			assert.Equal(t, test.want.string, buf.String())
		})
	}
}

func TestPayloadDecodeV4(t *testing.T) {
	type want struct {
		pay PayloadV4
		err error
	}
	var tests = []struct {
		name string
		data string
		want want
	}{
		{
			name: "no binary",
			data: "4hello\x1e4€",
			want: want{err: nil, pay: []PacketV4{
				{Packet: Packet{T: MessagePacket, D: "hello"}},
				{Packet: Packet{T: MessagePacket, D: "€"}},
			}},
		},
		{
			name: "with binary",
			data: "4€\x1ebAQIDBA==",
			want: want{err: nil, pay: []PacketV4{
				{Packet: Packet{T: MessagePacket, D: "€"}},
				{Packet: Packet{T: BinaryPacket, D: []byte{1, 2, 3, 4}}, IsBinary: true},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			var pay PayloadV4
			err := NewPayloadDecoderV4(strings.NewReader(test.data)).Decode(&pay)

			assert.Equal(t, test.want.err, err)
			assert.Equal(t, test.want.pay, pay)
		})
	}
}
