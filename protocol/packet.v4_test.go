package protocol

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketV4(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PacketV4, [][]byte, error) testFn
		testParamsOutFn func(*testing.T) (PacketV4, [][]byte, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(input PacketV4, output [][]byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				{
					n, err := have.ReadFrom(&input)
					assert.Equal(t, int64(len(output[0])), n)
					assert.Equal(t, xErr, err)
				}

				assert.Equal(t, output[0], have.Bytes())
				for i, next := range output[1:] {
					have.Reset()
					input.outgoing.WriteTo(have)
					assert.Equal(t, next, have.Bytes(), fmt.Sprintf("writeTo #%d", i))
					cont := input.outgoing.Next()
					assert.Equal(t, i != len(output[1:])-1, cont, fmt.Sprintf("cont #%d", i))
				}
			}
		},
		"Write": func(input PacketV4, output [][]byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PacketV4
				n, err := (&have).Write(output[0])

				assert.Equal(t, len(output[0]), n, "the data length")
				assert.ErrorIs(t, err, xErr)

				go func() {
					for i, next := range have.incoming {
						err := next(bytes.NewReader(output[i+1]))
						assert.NoError(t, err)
					}
				}()

				assert.Equal(t, input.Type, have.Type, "the packet type")
				assert.Equal(t, input.Namespace, have.Namespace, "the namespace:")
				assert.Equal(t, input.AckID, have.AckID, "the ackID")
				assert.IsType(t, input.Data, have.Data, "the data (type)")

				switch _have := have.Data.(type) {
				case *packetDataArray:
					for i, wantx := range input.Data.(*packetDataArray).x {
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
		"WriteTo": func(input PacketV4, output [][]byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				{
					n, err := input.WriteTo(have)
					assert.Equal(t, int64(len(output[0])), n)
					assert.Equal(t, xErr, err)
				}

				assert.Equal(t, output[0], have.Bytes())
				for i, next := range output[1:] {
					have.Reset()
					input.outgoing.WriteTo(have)
					assert.Equal(t, next, have.Bytes(), fmt.Sprintf("writeTo #%d", i))
					cont := input.outgoing.Next()
					assert.Equal(t, i != len(output[1:])-1, cont, fmt.Sprintf("cont #%d", i))
				}
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`0`)}
			asPacket := *NewPacketV4().WithNamespace("/").(*PacketV4)
			return asPacket, asBytes, nil
		},
		"DISCONNECT": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`1/admin`)}
			asPacket := *NewPacketV4().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV4)
			return asPacket, asBytes, nil
		},
		"EVENT": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`2["hello",1]`)}
			asPacket := *NewPacketV4().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV4)
			return asPacket, asBytes, nil
		},
		"EVENT with AckID": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`2/admin,456["project:delete",123]`)}
			asPacket := *NewPacketV4().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV4)
			return asPacket, asBytes, nil
		},
		"ACK": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`3/admin,456[]`)}
			asPacket := *NewPacketV4().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV4)
			return asPacket, asBytes, nil
		},
		"ERROR": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`4/admin,"Not authorized"`)}
			asPacket := *NewPacketV4().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV4)
			return asPacket, asBytes, nil
		},
		"BINARY EVENT": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{
				[]byte(`51-["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			asPacket := *NewPacketV4().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV4)
			return asPacket, asBytes, nil
		},
		"BINARY EVENT with AckID": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{
				[]byte(`51-/admin,456["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			asPacket := *NewPacketV4().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).
				WithAckID(456).(*PacketV4)
			return asPacket, asBytes, nil
		},
		"BINARY ACK": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{
				[]byte(`61-/admin,456[{"_placeholder":true,"num":0}]`),
				{0x03, 0x02, 0x01},
			}
			asPacket := *NewPacketV4().
				WithType(BinaryAckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{bytes.NewReader([]byte{0x03, 0x02, 0x01})}).
				WithAckID(456).(*PacketV4)
			return asPacket, asBytes, nil
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`0/admin`)}
			asPacket := *NewPacketV4().
				WithNamespace("/admin").(*PacketV4)
			return asPacket, asBytes, nil
		},
		"CONNECT /admin ns and extra info": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{[]byte(`0/admin?token=1234&uid=abcd`)}
			asPacket := *NewPacketV4().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV4)
			return asPacket, asBytes, nil
		},
		"BINARY ACK with /admin ns": func(*testing.T) (PacketV4, [][]byte, error) {
			asBytes := [][]byte{
				[]byte(`61-/admin,456{"stream":{"_placeholder":true,"num":0}}`),
				{0x03, 0x02, 0x01},
			}
			asPacket := *NewPacketV4().
				WithType(BinaryAckPacket.Byte()).
				WithNamespace("/admin").
				WithData(map[string]interface{}{"stream": bytes.NewReader([]byte{0x03, 0x02, 0x01})}).
				WithAckID(456).(*PacketV4)
			return asPacket, asBytes, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
