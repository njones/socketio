package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketV2Read(ext *testing.T) {
	type want struct {
		data [][]byte
		n    int64
		err  error
	}

	var tests = []struct {
		name string
		pac  PacketV2
		want want
	}{
		{
			name: "CONNECT",
			pac:  PacketV2{packet: packet{Type: ConnectPacket, Namespace: packetNS("/")}},
			want: want{
				data: [][]byte{[]byte(`0`)},
				n:    1,
				err:  nil,
			},
		},
		{
			name: "CONNECT /admin ns",
			pac:  PacketV2{packet: packet{Type: ConnectPacket, Namespace: packetNS("/admin")}},
			want: want{
				data: [][]byte{[]byte(`0/admin`)},
				n:    7,
				err:  nil,
			},
		},
		{
			name: "EVENT",
			pac:  PacketV2{packet: packet{Type: EventPacket, Namespace: packetNS("/"), Data: &packetDataArray{skipBinary: true, x: []interface{}{"hello", 1.0}}}},
			want: want{
				data: [][]byte{[]byte(`2["hello",1]`)},
				n:    12,
				err:  nil,
			},
		},
		{
			name: "EVENT with AckID",
			pac:  PacketV2{packet: packet{Type: EventPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{skipBinary: true, x: []interface{}{"project:delete", 123.0}}, AckID: 456}},
			want: want{
				data: [][]byte{[]byte(`2/admin,456["project:delete",123]`)},
				n:    33,
				err:  nil,
			},
		},
		{
			name: "ACK",
			pac:  PacketV2{packet: packet{Type: AckPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{skipBinary: true}, AckID: 456}},
			want: want{
				data: [][]byte{[]byte(`3/admin,456[]`)},
				n:    13,
				err:  nil,
			},
		},
		{
			name: "ERROR",
			pac:  PacketV2{packet: packet{Type: ErrorPacket, Namespace: packetNS("/admin"), Data: &packetDataString{&noAuth}}},
			want: want{
				data: [][]byte{[]byte(`4/admin,"Not authorized"`)},
				n:    24,
				err:  nil,
			},
		},
		{
			name: "CONNECT /admin ns and extra info",
			pac:  PacketV2{packet: packet{Type: ConnectPacket, Namespace: packetNS("/admin?token=1234&uid=abcd")}},
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
			(&test.pac).init()
			n, err := have[0].ReadFrom(&test.pac)

			assert.Equal(t, test.want.n, n)
			assert.Equal(t, test.want.err, err)

			if assert.Equal(t, len(test.want.data), len(have), "the buffers should match the data that we have.") {
				for i, want := range test.want.data {
					assert.Equal(t, want, have[i].Bytes())
				}
			}

		})
	}
}

func TestPacketV2Write(ext *testing.T) {
	type want struct {
		pac PacketV2
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
				pac: PacketV2{packet: packet{Type: ConnectPacket, Namespace: packetNS("/")}},
				n:   1,
				err: nil,
			},
		},
		{
			name: "CONNECT /admin ns",
			data: [][]byte{[]byte("0/admin")},
			want: want{
				pac: PacketV2{packet: packet{Type: ConnectPacket, Namespace: packetNS("/admin")}},
				n:   7,
				err: nil,
			},
		},
		// {
		// 	name: "EVENT",
		// 	data: [][]byte{[]byte(`2["hello",1]`)},
		// 	want: want{
		// 		pac: PacketV2{packet: packet{Type: EventPacket, Namespace: packetNS("/"), Data: &packetDataArray{skipBinary: true, x: []interface{}{"hello", 1.0}}}},
		// 		n:   12,
		// 		err: nil,
		// 	},
		// },
		// {
		// 	name: "EVENT with AckID",
		// 	data: [][]byte{[]byte(`2/admin,456["project:delete",123]`)},
		// 	want: want{
		// 		pac: PacketV2{packet: packet{Type: EventPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{skipBinary: true, x: []interface{}{"project:delete", 123.0}}, AckID: 456}},
		// 		n:   33,
		// 		err: nil,
		// 	},
		// },
		// {
		// 	name: "ACK",
		// 	data: [][]byte{[]byte(`3/admin,456[]`)},
		// 	want: want{
		// 		pac: PacketV2{packet: packet{Type: AckPacket, Namespace: packetNS("/admin"), Data: &packetDataArray{skipBinary: true, x: []interface{}{}}, AckID: 456}},
		// 		n:   13,
		// 		err: nil,
		// 	},
		// },
		// {
		// 	name: "ERROR",
		// 	data: [][]byte{[]byte(`4/admin,"Not authorized"`)},
		// 	want: want{
		// 		pac: PacketV2{packet: packet{Type: ErrorPacket, Namespace: packetNS("/admin"), Data: &packetDataString{&noAuth}}},
		// 		n:   24,
		// 		err: nil,
		// 	},
		// },
	}

	for _, test := range tests {
		ext.Run(test.name, func(t *testing.T) {
			var pac PacketV2
			n, err := (&pac).Write(test.data[0])

			assert.Equal(t, test.want.n, n)
			assert.ErrorIs(t, err, test.want.err)

			assert.Equal(t, test.want.pac.Type, pac.Type)
			assert.Equal(t, test.want.pac.Namespace, pac.Namespace)
			assert.Equal(t, test.want.pac.AckID, pac.AckID)

			if len(test.data[1:]) == 0 {
				assert.Equal(t, test.want.pac.Data, pac.Data)
			}
		})
	}
}
