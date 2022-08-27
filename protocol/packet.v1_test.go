package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	itst "github.com/njones/socketio/internal/test"
	"github.com/stretchr/testify/assert"
)

var notAuthorized = "Not authorized"
var runTest, skipTest = itst.RunTest, itst.SkipTest

func TestPacketV1(t *testing.T) {
	var opts = []func(*testing.T) bool{}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(PacketV1, []byte, error) testFn
		testParamsOutFn func(*testing.T) (PacketV1, []byte, error)
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(input PacketV1, output []byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				var have = new(bytes.Buffer)
				n, err := have.ReadFrom(&input)
				assert.Equal(t, int64(len(output)), n)
				assert.Equal(t, xErr, err)

				assert.Equal(t, output, have.Bytes())
			}
		},
		"Write": func(output PacketV1, input []byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				var packet PacketV1
				n, err := (&packet).Write(input)

				assert.Equal(t, len(input), n, "the data length")
				assert.ErrorIs(t, err, xErr)

				assert.Equal(t, output.Type, packet.Type, "the packet type")
				assert.Equal(t, output.Namespace, packet.Namespace, "the namespace:")
				assert.Equal(t, output.AckID, packet.AckID, "the ackID")
			}
		},
		"WriteTo": func(input PacketV1, output []byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					if !opt(t) {
						return
					}
				}

				var have = new(bytes.Buffer)
				n, err := input.WriteTo(have)
				assert.Equal(t, int64(len(output)), n)
				assert.Equal(t, xErr, err)

				assert.Equal(t, output, have.Bytes())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`0`)
			asPacket := *NewPacketV1().WithNamespace("/").(*PacketV1)
			return asPacket, asBytes, nil
		},
		"DISCONNECT": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`1/admin`)
			asPacket := *NewPacketV1().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV1)
			return asPacket, asBytes, nil
		},
		"EVENT": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`2["hello",1]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV1)
			return asPacket, asBytes, nil
		},
		"EVENT with AckID": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`2/admin,456["project:delete",123]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV1)
			return asPacket, asBytes, nil
		},
		"ACK": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`3/admin,456[]`)
			asPacket := *NewPacketV1().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV1)
			return asPacket, asBytes, nil
		},
		"ERROR": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`4/admin,"Not authorized"`)
			asPacket := *NewPacketV1().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV1)
			return asPacket, asBytes, nil
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`0/admin`)
			asPacket := *NewPacketV1().
				WithNamespace("/admin").(*PacketV1)
			return asPacket, asBytes, nil
		},
		"CONNECT /admin ns and extra info": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`0/admin?token=1234&uid=abcd`)
			asPacket := *NewPacketV1().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV1)
			return asPacket, asBytes, nil
		},
		"EVENT with Binary": func(t2 *testing.T) (PacketV1, []byte, error) {
			opts = append(opts, itst.DoNotTest("Write"))

			asError := ErrBinaryDataUnsupported
			asBytes := []byte(`2[]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{strings.NewReader("binary data")}).(*PacketV1)

			return asPacket, asBytes, asError
		},
		"EVENT with Binary unsupported": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`2["unsupported",{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{
					"unsupported",
					map[string]interface{}{"base64": true, "data": "xAtiaW5hcnkgZGF0YQ=="},
				}).(*PacketV1)
			return asPacket, asBytes, nil
		},
		"EVENT with Object": func(*testing.T) (PacketV1, []byte, error) {
			asBytes := []byte(`2["hello",{"playground":"world","wake":{"won":["too",3]}}]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", map[string]interface{}{
					"playground": "world",
					"wake": map[string]interface{}{
						"won": []interface{}{"too", 3},
					},
				}}).(*PacketV1)
			return asPacket, asBytes, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
