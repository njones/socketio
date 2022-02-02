package protocol

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPacketV3DecodingSadPath(t *testing.T) {

	var tests = []struct {
		name string
		bBin bool
		data io.Reader
		want error
	}{
		{
			name: "[open]",
			data: strings.NewReader(`0{"sid":`),
			want: ErrHandshakeDecode,
		},
		{
			name: "[unknown]",
			data: &sadReadWriter{data: []byte(`x`), err: fmt.Errorf("bad connection")},
			want: ErrPacketDecode,
		},
		{
			name: "[message]",
			data: &sadReadWriter{data: []byte(`4Hello`), err: fmt.Errorf("bad connection")},
			want: ErrPacketDecode,
		},
		{
			name: "[message] binary",
			bBin: true,
			data: &sadReadWriter{data: []byte(`4Hello`), err: fmt.Errorf("bad connection")},
			want: ErrPacketDecode,
		},
		{
			name: "[ping/pong]",
			data: &sadReadWriter{data: []byte(`2Hello`), err: fmt.Errorf("bad connection")},
			want: ErrPacketDecode,
		},
		{
			name: "[message] skip sad on io.EOF",
			data: &sadReadWriter{data: []byte(`4HelloWorld`), err: io.EOF},
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t2 *testing.T) {
			err := NewPacketDecoderV3(test.data).Decode(&PacketV3{IsBinary: test.bBin})
			assert.ErrorIs(t2, err, test.want)
		})
	}
}

func TestPacketV3Decoding(t *testing.T) {
	type want struct {
		PacketV3
		error
	}
	var tests = []struct {
		name string
		data string
		null bool
		want want
	}{
		{
			name: "[open]",
			data: `0{"sid":"abc123","upgrades":[],"pingTimeout":5000}`,
			null: true,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: OpenPacket,
				D: HandshakeV3{HandshakeV2: HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(5000 * time.Millisecond)}},
			}}},
		},
		{
			name: "[close]",
			data: `1`,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: ClosePacket,
				D: nil,
			}}},
		},
		{
			name: "[ping]",
			data: `2`,
			null: true,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: PingPacket,
				D: nil,
			}}},
		},
		{
			name: "[pong]",
			data: `3probe`,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: PongPacket,
				D: "probe",
			}}},
		},
		{
			name: "[message]",
			data: `4HelloWorld`,
			null: true,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: MessagePacket,
				D: "HelloWorld",
			}}},
		},
		{
			name: "[upgrade]",
			data: `5`,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: UpgradePacket,
				D: nil,
			}}},
		},
		{
			name: "[noop]",
			data: `6`,
			null: true,
			want: want{error: nil, PacketV3: PacketV3{Packet: Packet{
				T: NoopPacket,
				D: nil,
			}}},
		},
	}

	var decoder _packetDecoderV3 = NewPacketDecoderV3

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf = strings.NewReader(test.data)
			{
				var pkt PacketV3
				if !test.null {
					pkt = PacketV3{} // not nil
				}
				err := NewPacketDecoderV3(buf).Decode(&pkt)

				assert.ErrorIs(t, err, test.want.error)
				assert.Equal(t, test.want.PacketV3, pkt)
			}

			buf.Reset(test.data)
			{
				var pkt PacketV3
				if !test.null {
					pkt = PacketV3{} // not nil
				}
				err := decoder.From(buf).ReadPacket(&pkt)

				assert.ErrorIs(t, err, test.want.error)
				assert.Equal(t, test.want.PacketV3, pkt)
			}
		})
	}
}

func TestPacketV3Encoding(t *testing.T) {
	type writerTo string
	type want struct {
		string
		error
	}
	var tests = []struct {
		name string
		kind PacketType
		data interface{}
		want want
	}{
		{
			name: "[open] no upgrades.",
			kind: OpenPacket,
			data: HandshakeV3{HandshakeV2: HandshakeV2{SID: "abc123", PingTimeout: Duration(5000 * time.Millisecond)}, PingInterval: Duration(5000 * time.Millisecond)},
			want: want{error: nil, string: `0{"sid":"abc123","upgrades":[],"pingTimeout":5000,"pingInterval":5000}`},
		},
		{
			name: "[open] with upgrades",
			kind: OpenPacket,
			data: HandshakeV3{HandshakeV2: HandshakeV2{SID: "abc123", Upgrades: []string{"polling"}, PingTimeout: Duration(5000 * time.Millisecond)}, PingInterval: Duration(5000 * time.Millisecond)},
			want: want{error: nil, string: `0{"sid":"abc123","upgrades":["polling"],"pingTimeout":5000,"pingInterval":5000}`},
		},
		{
			name: "[close]",
			kind: ClosePacket,
			want: want{error: nil, string: `1`},
		},
		{
			name: "[ping]",
			kind: PingPacket,
			want: want{error: nil, string: `2`},
		},
		{
			name: "[pong]",
			kind: PongPacket,
			data: "probe",
			want: want{error: nil, string: `3probe`},
		},
		{
			name: "[message]",
			kind: MessagePacket,
			data: "HelloWorld",
			want: want{error: nil, string: `4HelloWorld`},
		},
		{
			name: "[message]",
			kind: MessagePacket,
			data: writerTo("HelloWorld"),
			want: want{error: nil, string: `4HelloWorld`},
		},
		{
			name: "[upgrade]",
			kind: UpgradePacket,
			want: want{error: nil, string: `5`},
		},
		{
			name: "[noop]",
			kind: NoopPacket,
			want: want{error: nil, string: `6`},
		},
		{
			name: "[binary]",
			kind: PacketType(20),
			want: want{error: ErrInvalidPacketType, string: ``},
		},
	}

	var encoder _packetEncoderV3 = NewPacketEncoderV3

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf = new(bytes.Buffer)
			{
				var data = test.data
				if _data, ok := test.data.(writerTo); ok {
					data = strings.NewReader(string(_data))
				}
				var pkt = Packet{T: test.kind, D: data}
				var err = NewPacketEncoderV3(buf).Encode(PacketV3{Packet: pkt})

				assert.ErrorIs(t, err, test.want.error)
				assert.Equal(t, test.want.string, buf.String())
			}

			buf.Reset()
			{
				var data = test.data
				if _data, ok := test.data.(writerTo); ok {
					data = strings.NewReader(string(_data))
				}
				var pkt = Packet{T: test.kind, D: data}
				var err = encoder.To(buf).WritePacket(PacketV3{Packet: pkt})

				assert.ErrorIs(t, err, test.want.error)
				assert.Equal(t, test.want.string, buf.String())
			}
		})
	}
}
