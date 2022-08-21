package protocol

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketV5Read(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(pac PacketV5, want [][]byte, xerr error) testFn
		testParamsOutFn func(*testing.T) (pac PacketV5, want [][]byte, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(pac PacketV5, want [][]byte, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := have.ReadFrom(&pac)
				assert.Equal(t, int64(len(want[0])), n)
				assert.Equal(t, xerr, err)

				assert.Equal(t, want[0], have.Bytes())
				for i, next := range want[1:] {
					have.Reset()
					pac.outgoing.WriteTo(have)
					assert.Equal(t, next, have.Bytes(), fmt.Sprintf("writeTo #%d", i))
					cont := pac.outgoing.Next()
					assert.Equal(t, i != len(want[1:])-1, cont, fmt.Sprintf("cont #%d", i))
				}
			}
		},
		"WriteTo": func(pac PacketV5, want [][]byte, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := pac.WriteTo(have)
				assert.Equal(t, int64(len(want[0])), n)
				assert.Equal(t, xerr, err)

				assert.Equal(t, want[0], have.Bytes())
				for i, next := range want[1:] {
					have.Reset()
					pac.outgoing.WriteTo(have)
					assert.Equal(t, next, have.Bytes(), fmt.Sprintf("writeTo #%d", i))
					cont := pac.outgoing.Next()
					assert.Equal(t, i != len(want[1:])-1, cont, fmt.Sprintf("cont #%d", i))
				}
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`0`)}
			data := *NewPacketV5().(*PacketV5)
			return data, want, nil
		},
		"CONNECT with Payload": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`0{"token":"123"}`)}
			data := *NewPacketV5().
				WithData(map[string]interface{}{"token": "123"}).(*PacketV5)
			return data, want, nil
		},
		"CONNECT /admin with Payload": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`0/admin,{"token":"123"}`)}
			data := *NewPacketV5().
				WithNamespace("/admin").
				WithData(map[string]interface{}{"token": "123"}).(*PacketV5)
			return data, want, nil
		},
		"DISCONNECT": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`1/admin`)}
			data := *NewPacketV5().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV5)
			return data, want, nil
		},
		"EVENT": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`2["hello",1]`)}
			data := *NewPacketV5().
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"hello", 1.0}).(*PacketV5)
			return data, want, nil
		},
		"EVENT with AckID": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`2/admin,456["project:delete",123]`)}
			data := *NewPacketV5().
				WithNamespace("/admin").
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},
		"ACK": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`3/admin,456[]`)}
			data := *NewPacketV5().
				WithNamespace("/admin").
				WithType(AckPacket.Byte()).
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},
		"ERROR": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`4/admin,"Not authorized"`)}
			data := *NewPacketV5().
				WithNamespace("/admin").
				WithType(ErrorPacket.Byte()).
				WithData(&notAuthorized).(*PacketV5)
			return data, want, nil
		},
		"BINARY EVENT": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{
				[]byte(`51-["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			data := *NewPacketV5().
				WithType(BinaryEventPacket.Byte()).
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV5)
			return data, want, nil
		},
		"BINARY EVENT with AckID": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{
				[]byte(`51-/admin,456["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			data := *NewPacketV5().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},
		"BINARY ACK": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{
				[]byte(`61-/admin,456[{"_placeholder":true,"num":0}]`),
				{0x03, 0x02, 0x01},
			}
			data := *NewPacketV5().
				WithType(BinaryAckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{bytes.NewReader([]byte{0x03, 0x02, 0x01})}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`0/admin`)}
			data := *NewPacketV5().
				WithNamespace("/admin").(*PacketV5)
			return data, want, nil
		},
		"CONNECT /admin ns and extra info": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`0/admin?token=1234&uid=abcd`)}
			data := *NewPacketV5().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV5)
			return data, want, nil
		},
		"ACK no namespace": func(*testing.T) (PacketV5, [][]byte, error) {
			want := [][]byte{[]byte(`3456["make",123]`)}
			data := *NewPacketV5().
				WithType(AckPacket.Byte()).
				WithData([]interface{}{"make", 123}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePacketV5(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(want [][]byte, pac PacketV5, xerr error) testFn
		testParamsOutFn func(*testing.T) (want [][]byte, pac PacketV5, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Write": func(data [][]byte, want PacketV5, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PacketV5
				n, err := (&have).Write(data[0])

				assert.Equal(t, len(data[0]), n, "the data length")
				assert.ErrorIs(t, err, xerr)

				go func() {
					for i, next := range have.incoming {
						err := next(bytes.NewReader(data[i+1]))
						assert.NoError(t, err)
					}
				}()

				assert.Equal(t, want.Type, have.Type, "the packet type")
				assert.Equal(t, want.Namespace, have.Namespace, "the namespace:")
				assert.Equal(t, want.AckID, have.AckID, "the ackID")
				assert.IsType(t, want.Data, have.Data, "the data (type)")

				switch _have := have.Data.(type) {
				case *packetDataArray:
					for i, wantx := range want.Data.(*packetDataArray).x {
						switch wx := wantx.(type) {
						case io.Reader:
							assert.Implements(t, (*io.Reader)(nil), _have.x[i])
							hx := _have.x[i].(io.Reader)

							wantb, err := io.ReadAll(wx)
							assert.NoError(t, err)

							haveb, err := io.ReadAll(hx)
							assert.NoError(t, err)

							assert.Equal(t, wantb, haveb)
						default:
							assert.Equal(t, wantx, _have.x[i])
						}
					}
				}
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`0`)}
			want := *NewPacketV5().WithNamespace("/").(*PacketV5)
			return data, want, nil
		},
		"CONNECT with Payload": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`0{"token":"123"}`)}
			want := *NewPacketV5().
				WithNamespace("/").
				WithData(map[string]interface{}{"token": "123"}).(*PacketV5)
			return data, want, nil
		},
		"CONNECT /admin with Payload": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`0/admin,{"token":"123"}`)}
			want := *NewPacketV5().
				WithNamespace("/admin").
				WithData(map[string]interface{}{"token": "123"}).(*PacketV5)
			return data, want, nil
		},
		"DISCONNECT": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`1`)}
			want := *NewPacketV5().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/").(*PacketV5)
			return data, want, nil
		},
		"EVENT": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`2["hello",1]`)}
			want := *NewPacketV5().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV5)
			return data, want, nil
		},
		"EVENT with AckID": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`2/admin,456["project:delete",123]`)}
			want := *NewPacketV5().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},
		"ACK": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`3/admin,456[]`)}
			want := *NewPacketV5().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV5)
			return data, want, nil
		},
		"ERROR": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{[]byte(`4/admin,"Not authorized"`)}
			want := *NewPacketV5().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(notAuthorized).(*PacketV5)
			return data, want, nil
		},
		"BINARY EVENT": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{
				[]byte(`51-["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			want := *NewPacketV5().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV5)
			return data, want, nil
		},
		"BINARY EVENT with AckID": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{
				[]byte(`51-/admin,456["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			want := *NewPacketV5().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithAckID(456).
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV5)
			return data, want, nil
		},
		"BINARY ACK": func(*testing.T) ([][]byte, PacketV5, error) {
			data := [][]byte{
				[]byte(`61-/admin,456[{"_placeholder":true,"num":0}]`),
				{0x03, 0x02, 0x01},
			}
			want := *NewPacketV5().
				WithType(BinaryAckPacket.Byte()).
				WithNamespace("/admin").
				WithAckID(456).
				WithData([]interface{}{bytes.NewReader([]byte{0x03, 0x02, 0x01})}).(*PacketV5)
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
