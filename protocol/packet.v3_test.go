package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

var noAuth = "Not authorized"

func TestPacketV3Read(ext *testing.T) {
	type want struct {
		data [][]byte
		n    int64
		err  error
	}

	var tests = []struct {
		name string
		pac  PacketV3
		want want
	}{
		{
			name: "CONNECT",
			pac:  PacketV3{packet: packet{Type: ConnectPacket, Namespace: packetNS("/")}},
			want: want{
				data: [][]byte{[]byte(`0`)},
				n:    1,
				err:  nil,
			},
		},
		{
			name: "CONNECT /admin ns",
			pac:  PacketV3{packet: packet{Type: ConnectPacket, Namespace: packetNS("/admin")}},
			want: want{
				data: [][]byte{[]byte(`0/admin`)},
				n:    7,
				err:  nil,
			},
		},
		{
			name: "EVENT",
			pac:  PacketV3{packet: packet{Type: EventPacket, Namespace: packetNS("/"), Data: &packetDataArray{x: []interface{}{"hello", 1.0}}}},
			want: want{
				data: [][]byte{[]byte(`2["hello",1]`)},
				n:    12,
				err:  nil,
			},
		},
		{
			name: "EVENT with AckID",
			pac:  PacketV3{packet: packet{Type: EventPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{x: []interface{}{"project:delete", 123.0}}, AckID: 456}},
			want: want{
				data: [][]byte{[]byte(`2/admin,456["project:delete",123]`)},
				n:    33,
				err:  nil,
			},
		},
		{
			name: "ACK",
			pac:  PacketV3{packet: packet{Type: AckPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{}, AckID: 456}},
			want: want{
				data: [][]byte{[]byte(`3/admin,456[]`)},
				n:    13,
				err:  nil,
			},
		},
		{
			name: "ERROR",
			pac:  PacketV3{packet: packet{Type: ErrorPacket, Namespace: packetNS("/admin"), Data: &packetDataString{&noAuth}}},
			want: want{
				data: [][]byte{[]byte(`4/admin,"Not authorized"`)},
				n:    24,
				err:  nil,
			},
		},
		{
			name: "BINARY EVENT",
			pac:  PacketV3{packet: packet{Type: BinaryEventPacket, Namespace: packetNS("/"), Data: &packetDataArray{x: []interface{}{"hello", bytes.NewReader([]byte{1, 2, 3})}}}},
			want: want{
				data: [][]byte{
					[]byte(`51-["hello",{"_placeholder":true,"num":0}]`),
					{1, 2, 3},
				},
				n:   42,
				err: nil,
			},
		},
		{
			name: "BINARY EVENT with Ack",
			pac:  PacketV3{packet: packet{Type: BinaryEventPacket, AckID: 456, Namespace: packetNS("/admin"), Data: &packetDataArray{x: []interface{}{"hello", bytes.NewReader([]byte{1, 2, 3})}}}},
			want: want{
				data: [][]byte{
					[]byte(`51-/admin,456["hello",{"_placeholder":true,"num":0}]`),
					{1, 2, 3},
				},
				n:   52,
				err: nil,
			},
		},
		{
			name: "CONNECT /admin ns and extra info",
			pac:  PacketV3{packet: packet{Type: ConnectPacket, Namespace: packetNS("/admin?token=1234&uid=abcd")}},
			want: want{
				data: [][]byte{[]byte(`0/admin?token=1234&uid=abcd`)},
				n:    27,
				err:  nil,
			},
		},
	}

	for _, test := range tests {
		var have []*bytes.Buffer
		have = append(have, new(bytes.Buffer))
		ext.Run(test.name, func(t *testing.T) {
			n, err := have[0].ReadFrom(&test.pac)

			assert.Equal(t, test.want.n, n)
			assert.Equal(t, test.want.err, err)

			for test.pac.outgoing.Next() {
				buf := new(bytes.Buffer)
				test.pac.outgoing.WriteTo(buf)
				have = append(have, buf)
			}

			if assert.Equal(t, len(test.want.data), len(have), "the buffers should match the data that we have.") {
				for i, want := range test.want.data {
					assert.Equal(t, want, have[i].Bytes())
				}
			}

		})
	}
}

func TestPacketV3Write(ext *testing.T) {
	type want struct {
		pac PacketV3
		n   int
		err error
	}

	var tests = []struct {
		name string
		data [][]byte
		want want
	}{
		{
			name: "CONNECT",
			data: [][]byte{[]byte("0")},
			want: want{
				pac: PacketV3{packet: packet{Type: ConnectPacket, Namespace: packetNS("/")}},
				n:   1,
				err: nil,
			},
		},
		{
			name: "CONNECT /admin ns",
			data: [][]byte{[]byte("0/admin")},
			want: want{
				pac: PacketV3{packet: packet{Type: ConnectPacket, Namespace: packetNS("/admin")}},
				n:   7,
				err: nil,
			},
		},
		{
			name: "EVENT",
			data: [][]byte{[]byte(`2["hello",1]`)},
			want: want{
				pac: PacketV3{packet: packet{Type: EventPacket, Namespace: packetNS("/"), Data: &packetDataArray{x: []interface{}{"hello", 1.0}}}},
				n:   12,
				err: nil,
			},
		},
		{
			name: "EVENT with AckID",
			data: [][]byte{[]byte(`2/admin,456["project:delete",123]`)},
			want: want{
				pac: PacketV3{packet: packet{Type: EventPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{x: []interface{}{"project:delete", 123.0}}, AckID: 456}},
				n:   33,
				err: nil,
			},
		},
		{
			name: "ACK",
			data: [][]byte{[]byte(`3/admin,456[]`)},
			want: want{
				pac: PacketV3{packet: packet{Type: AckPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{x: []interface{}{}}, AckID: 456}},
				n:   13,
				err: nil,
			},
		},
		{
			name: "ERROR",
			data: [][]byte{[]byte(`4/admin,"Not authorized"`)},
			want: want{
				pac: PacketV3{packet: packet{Type: ErrorPacket, Namespace: packetNS("/admin"), Data: &packetDataString{&noAuth}}},
				n:   24,
				err: nil,
			},
		},
		{
			name: "BINARY EVENT",
			data: [][]byte{
				[]byte(`51-["hello",{"_placeholder":true,"num":0}]`),
				{1, 2, 3},
			},
			want: want{
				pac: PacketV3{packet: packet{Type: BinaryEventPacket, Namespace: packetNS("/"), Data: &packetDataArray{x: []interface{}{"hello", io.Reader(nil)}}}, incoming: binaryStreamIn{func(io.Reader) error { return nil }}},
				n:   42,
				err: nil,
			},
		},
		{
			name: "BINARY EVENT with Ack",
			data: [][]byte{
				[]byte(`51-/admin,456["hello",{"_placeholder":true,"num":0}]`),
				{1, 2, 3},
			},
			want: want{
				pac: PacketV3{packet: packet{Type: BinaryEventPacket, AckID: 456, Namespace: packetNS("/admin"), Data: &packetDataArray{x: []interface{}{"hello", io.Reader(nil)}}}, incoming: binaryStreamIn{func(io.Reader) error { return nil }}},
				n:   52,
				err: nil,
			},
		},
	}

	for _, test := range tests {
		ext.Run(test.name, func(t *testing.T) {
			var pac PacketV3
			n, err := (&pac).Write(test.data[0])

			assert.Equal(t, len(test.data[1:]), len(pac.incoming))
			if len(test.data) > 0 {
				for i, bin := range test.data[1:] {
					go func(ii int, bb []byte) { pac.incoming[ii](bytes.NewReader(bb)) }(i, bin)
				}
				var n int
				var bufs = make([]*bytes.Buffer, len(test.data[1:]))
				var _array, ok = pac.Data.(*packetDataArray)
				if ok && _array != nil {
					for _, v := range _array.x {
						if r, ok := v.(io.Reader); ok {
							bufs[n] = new(bytes.Buffer)
							bufs[n].ReadFrom(r)
							n++
						}
					}
				}
				assert.Equal(t, len(test.data[1:]), n)
				for i, buf := range bufs {
					assert.Equal(t, test.data[i+1], buf.Bytes())
				}
			}

			assert.Equal(t, test.want.n, n)
			assert.Equal(t, test.want.err, err)

			assert.Equal(t, test.want.pac.Type, pac.Type)
			assert.Equal(t, test.want.pac.Namespace, pac.Namespace)
			assert.Equal(t, test.want.pac.AckID, pac.AckID)

			if len(test.data[1:]) == 0 {
				assert.Equal(t, test.want.pac.Data, pac.Data)
			}

		})
	}
}
