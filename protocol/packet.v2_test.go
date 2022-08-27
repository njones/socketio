package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PacketV2, []byte, error) testFn
		testParamsOutFn func(*testing.T) (PacketV2, []byte, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(input PacketV2, output []byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := have.ReadFrom(&input)
				assert.Equal(t, int64(len(output)), n)
				assert.Equal(t, xErr, err)

				assert.Equal(t, output, have.Bytes())
			}
		},
		"Write": func(output PacketV2, input []byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PacketV2
				n, err := (&have).Write(input)

				assert.Equal(t, len(input), n, "the data length")
				assert.ErrorIs(t, err, xErr)

				assert.Equal(t, output.Type, have.Type, "the packet type")
				assert.Equal(t, output.Namespace, have.Namespace, "the namespace:")
				assert.Equal(t, output.AckID, have.AckID, "the ackID")
				assert.IsType(t, output.Data, have.Data, "the data (type)")

				switch _have := have.Data.(type) {
				case *packetDataArray:
					for i, _want := range output.Data.(*packetDataArray).x {
						assert.Equal(t, _want, _have.x[i])
					}
				}
			}
		},
		"WriteTo": func(input PacketV2, output []byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := (&input).WriteTo(have)
				assert.Equal(t, int64(len(output)), n)
				assert.Equal(t, xErr, err)

				assert.Equal(t, output, have.Bytes())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`0`)
			asPacket := *NewPacketV2().WithNamespace("/").(*PacketV2)
			return asPacket, asBytes, nil
		},
		"DISCONNECT": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`1/admin`)
			asPacket := *NewPacketV2().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV2)
			return asPacket, asBytes, nil
		},
		"EVENT": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`2["hello",1]`)
			asPacket := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"EVENT with AckID": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`2/admin,456["project:delete",123]`)
			asPacket := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"ACK": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`3/admin,456[]`)
			asPacket := *NewPacketV2().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"ERROR": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`4/admin,"Not authorized"`)
			asPacket := *NewPacketV2().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"BINARY EVENT": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`51-["hello",{"base64":true,"data":"xAMBAgM="}]`)
			asPacket := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"BINARY EVENT with AckID": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`51-/admin,456["hello",{"base64":true,"data":"xAMBAgM="}]`)
			asPacket := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).
				WithAckID(456).(*PacketV2)
			return asPacket, asBytes, nil
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`0/admin`)
			asPacket := *NewPacketV2().
				WithNamespace("/admin").(*PacketV2)
			return asPacket, asBytes, nil
		},
		"CONNECT /admin ns and extra info": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`0/admin?token=1234&uid=abcd`)
			asPacket := *NewPacketV2().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV2)
			return asPacket, asBytes, nil
		},
		"EVENT with Binary": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`21-["name",{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}]`)
			asPacket := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"name", bytes.NewReader([]byte("binary data"))}).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"ACK with Object": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`3/site,444{"api-mocked.com":"member"}`)
			asPacket := *NewPacketV2().
				WithType(AckPacket.Byte()).
				WithNamespace("/site").
				WithData(map[string]interface{}{"api-mocked.com": "member"}).
				WithAckID(444).(*PacketV2)
			return asPacket, asBytes, nil
		},
		"ACK with Object and Binary": func(*testing.T) (PacketV2, []byte, error) {
			asBytes := []byte(`31-/site,444{"api-mocked.com":{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}}`)
			asPacket := *NewPacketV2().
				WithType(AckPacket.Byte()).
				WithNamespace("/site").
				WithData(map[string]interface{}{"api-mocked.com": strings.NewReader("binary data")}).
				WithAckID(444).(*PacketV2)
			return asPacket, asBytes, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
