//go:build gc || (eio_pac_v2 && eio_pac_v3)

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

// sadReader will send an error after
// the data bytes have been sent
type sadReadWriter struct {
	n    int
	data []byte
	err  error

	idx  int
	errs []error
}

func (srw *sadReadWriter) Read(p []byte) (n int, err error) {
	n = copy(p, srw.data[srw.n:])
	srw.n += n

	if srw.n == len(srw.data) {
		srw.n = 0
		return 0, srw.err
	}

	return n, nil
}

func (srw *sadReadWriter) Write(p []byte) (n int, err error) {
	defer func() { srw.idx++ }()

	n = len(p)
	if len(srw.errs) > srw.idx {
		err = srw.errs[srw.idx]
		if err != nil {
			n = 0
		}
	}
	return n, err
}

func TestPacketV2DecodingSadPath(t *testing.T) {

	var tests = []struct {
		name string
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
			var pkt PacketV2

			err := NewPacketDecoderV2(test.data).Decode(&pkt)

			assert.ErrorIs(t2, err, test.want)
		})
	}
}

func TestPacketV2Decoding(t *testing.T) {
	type want struct {
		PacketV2
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
			data: `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`,
			null: true,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: OpenPacket,
				D: HandshakeV2{SID: "abc123", Upgrades: []string{}, PingTimeout: Duration(300 * time.Millisecond)},
			}}},
		},
		{
			name: "[close]",
			data: `1`,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: ClosePacket,
				D: nil,
			}}},
		},
		{
			name: "[ping]",
			data: `2`,
			null: true,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: PingPacket,
				D: nil,
			}}},
		},
		{
			name: "[pong]",
			data: `3probe`,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: PongPacket,
				D: "probe",
			}}},
		},
		{
			name: "[message]",
			data: `4HelloWorld`,
			null: true,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: MessagePacket,
				D: "HelloWorld",
			}}},
		},
		{
			name: "[upgrade]",
			data: `5`,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: UpgradePacket,
				D: nil,
			}}},
		},
		{
			name: "[noop]",
			data: `6`,
			null: true,
			want: want{error: nil, PacketV2: PacketV2{Packet: Packet{
				T: NoopPacket,
				D: nil,
			}}},
		},
	}

	var decoder _packetDecoderV2 = NewPacketDecoderV2

	for _, test := range tests {
		t.Run(test.name, func(t2 *testing.T) {
			var buf = strings.NewReader(test.data)
			{
				var pkt PacketV2

				if !test.null {
					pkt = PacketV2{} // not nil
				}

				err := NewPacketDecoderV2(buf).Decode(&pkt)

				assert.ErrorIs(t2, err, test.want.error)
				assert.Equal(t2, test.want.PacketV2, pkt)
			}

			buf.Reset(test.data)
			{
				var pkt PacketV2

				if !test.null {
					pkt = PacketV2{} // not nil
				}

				err := decoder.From(buf).ReadPacket(&pkt)

				assert.Equal(t2, test.want.error, err)
				assert.Equal(t2, test.want.PacketV2, pkt)
			}
		})
	}
}

func TestPacketV2EncodingSadPath(t *testing.T) {
	var tests = []struct {
		name   string
		data   PacketV2
		writer io.Writer
		want   error
	}{
		{
			name:   "[handshake] no data",
			data:   PacketV2{Packet: Packet{T: OpenPacket}},
			writer: &sadReadWriter{},
			want:   ErrInvalidHandshake,
		},
		{
			name:   "[handshake] 1",
			data:   PacketV2{Packet: Packet{T: OpenPacket, D: &HandshakeV2{}}},
			writer: &sadReadWriter{errs: []error{ErrHandshakeEncode}},
			want:   ErrHandshakeEncode,
		},
		{
			name:   "[handshake] 2",
			data:   PacketV2{Packet: Packet{T: OpenPacket, D: &HandshakeV2{}}},
			writer: &sadReadWriter{errs: []error{nil, ErrHandshakeEncode}},
			want:   ErrHandshakeEncode,
		},
		{
			name:   "[message] nil",
			data:   PacketV2{Packet: Packet{T: MessagePacket, D: nil}},
			writer: &sadReadWriter{errs: []error{ErrPacketEncode}},
			want:   ErrPacketEncode,
		},
		{
			name:   "[message] string",
			data:   PacketV2{Packet: Packet{T: MessagePacket, D: ""}},
			writer: &sadReadWriter{errs: []error{ErrPacketEncode}},
			want:   ErrPacketEncode,
		},
		{
			name:   "[message] io.WriterTo 1",
			data:   PacketV2{Packet: Packet{T: MessagePacket, D: strings.NewReader("A")}},
			writer: &sadReadWriter{errs: []error{ErrPacketEncode}},
			want:   ErrPacketEncode,
		},
		{
			name:   "[message] io.WriterTo 2",
			data:   PacketV2{Packet: Packet{T: MessagePacket, D: strings.NewReader("Sad")}},
			writer: &sadReadWriter{errs: []error{nil, ErrPacketEncode}},
			want:   ErrPacketEncode,
		},
		{
			name:   "[message] invalid packet data",
			data:   PacketV2{Packet: Packet{T: MessagePacket, D: 123}},
			writer: &sadReadWriter{},
			want:   ErrInvalidPacketData,
		},
		{
			name:   "[close]",
			data:   PacketV2{Packet: Packet{T: ClosePacket}},
			writer: &sadReadWriter{errs: []error{ErrPacketEncode}},
			want:   ErrPacketEncode,
		},
		{
			name:   "[invalid packet]",
			data:   PacketV2{Packet: Packet{T: 50}},
			writer: &sadReadWriter{},
			want:   ErrInvalidPacketType,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t2 *testing.T) {

			err := NewPacketEncoderV2(test.writer).Encode(test.data)

			assert.ErrorIs(t2, err, test.want)
		})
	}
}

func TestPacketV2Encoding(t *testing.T) {
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
			name: "[open] no upgrades",
			kind: OpenPacket,
			data: &HandshakeV2{SID: "abc123", PingTimeout: Duration(300 * time.Millisecond)},
			want: want{error: nil, string: `0{"sid":"abc123","upgrades":[],"pingTimeout":300}`},
		},
		{
			name: "[open] with upgrades",
			kind: OpenPacket,
			data: &HandshakeV2{SID: "abc123", Upgrades: []string{"polling"}, PingTimeout: Duration(300 * time.Millisecond)},
			want: want{error: nil, string: `0{"sid":"abc123","upgrades":["polling"],"pingTimeout":300}`},
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
			name: "[message] with io.WriterTo",
			kind: MessagePacket,
			data: writerTo("HelloWorld"), // will implment the WriterTo interface
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
			kind: BinaryPacket,
			want: want{error: ErrInvalidPacketType, string: ``},
		},
	}

	var encoder _packetEncoderV2 = NewPacketEncoderV2

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf = new(bytes.Buffer)
			{
				var data = test.data
				if _data, ok := test.data.(writerTo); ok {
					data = strings.NewReader(string(_data))
				}
				var pkt = Packet{T: test.kind, D: data}
				var err = NewPacketEncoderV2(buf).Encode(PacketV2{pkt})

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
				var err = encoder.To(buf).WritePacket(PacketV2{pkt})

				assert.ErrorIs(t, err, test.want.error)
				assert.Equal(t, test.want.string, buf.String())
			}
		})
	}
}
