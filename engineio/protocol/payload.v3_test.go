package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayloadEncodeV3(t *testing.T) {
	type want struct {
		str  string
		err  error
		xhr2 bool
	}
	var tests = []struct {
		name string
		data PayloadV3
		want want
	}{
		{
			name: "basic v2",
			data: []PacketV3{
				{Packet: Packet{T: PingPacket, D: "probe"}},
				{Packet: Packet{T: MessagePacket, D: "HelloWorld"}},
				{Packet: Packet{T: UpgradePacket}},
			},
			want: want{err: nil, str: `6:2probe11:4HelloWorld1:5`},
		},
		{
			name: "basic v3",
			data: []PacketV3{
				{Packet: Packet{T: MessagePacket, D: "hello"}},
				{Packet: Packet{T: MessagePacket, D: "€"}},
			},
			want: want{err: nil, str: `6:4hello2:4€`},
		},
		{
			name: "no binary transport",
			data: []PacketV3{
				{Packet: Packet{T: MessagePacket, D: "€"}},
				{Packet: Packet{T: MessagePacket, D: []byte{1, 2, 3, 4}}},
			},
			want: want{err: nil, str: `2:4€10:b4AQIDBA==`},
		},
		// {
		// 	name: "binary transport",
		// 	data: []PacketV3{
		// 		{Packet: Packet{T: MessagePacket, D: "€"}},
		// 		{Packet: Packet{T: MessagePacket, D: []byte{1, 2, 3, 4}}},
		// 	},
		// 	want: want{err: nil, str: "\x00\x04\xff\x34\xe2\x82\xac\x01\x04\xff\x01\x02\x03\x04", xhr2: true},
		// },
	}

	buf := new(bytes.Buffer)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()

			enc := NewPayloadEncoderV3(buf)
			if test.want.xhr2 {
				enc.IsXHR2 = true
			}
			err := enc.Encode(test.data)

			assert.Equal(t, test.want.err, err)
			assert.Equal(t, test.want.str, buf.String())
		})
	}
}

func TestPayloadDecodeV3(t *testing.T) {
	type want struct {
		pay  PayloadV3
		err  error
		xhr2 bool
	}
	var tests = []struct {
		name string
		data string
		want want
	}{
		{
			name: "basic v2",
			data: `6:2probe11:4HelloWorld1:5`,
			want: want{err: nil, pay: []PacketV3{
				{Packet: Packet{T: PingPacket, D: "probe"}},
				{Packet: Packet{T: MessagePacket, D: "HelloWorld"}},
				{Packet: Packet{T: UpgradePacket}},
			}},
		},
		{
			name: "basic v3",
			data: `6:4hello2:4€`,
			want: want{err: nil, pay: []PacketV3{
				{Packet: Packet{T: MessagePacket, D: "hello"}},
				{Packet: Packet{T: MessagePacket, D: "€"}},
			}},
		},
		{
			name: "no binary transport",
			data: `2:4€10:b4AQIDBA==`,
			want: want{err: nil, pay: []PacketV3{
				{Packet: Packet{T: MessagePacket, D: "€"}, IsBinary: false},
				{Packet: Packet{T: MessagePacket, D: []byte{1, 2, 3, 4}}, IsBinary: true},
			}},
		},
		// {
		// 	name: "binary transport",
		// 	data: "\x00\x04\xff\x34\xe2\x82\xac\x01\x04\xff\x01\x02\x03\x04",
		// 	want: want{err: nil, xhr2: true, pay: []PacketV3{
		// 		{Packet: Packet{T: MessagePacket, D: "€"}},
		// 		{Packet: Packet{T: MessagePacket, D: []byte{1, 2, 3, 4}}, IsBinary: true},
		// 	}},
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			var pay PayloadV3
			dec := NewPayloadDecoderV3(strings.NewReader(test.data))
			if test.want.xhr2 {
				dec.IsXHR2 = true
			}
			err := dec.Decode(&pay)

			assert.ErrorIs(t, test.want.err, err)
			assert.Equal(t, test.want.pay, pay)
		})
	}
}
