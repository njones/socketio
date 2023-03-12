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

type testData struct {
	rawBytes     []byte
	rawByteSlice [][]byte
	rawPacket    interface{}
	altPacket    interface{}
	err          error
}

func (d testData) rawPacketV1() PacketV1 { return d.rawPacket.(PacketV1) }
func (d testData) altPacketV1() PacketV1 { return (d.altPacket).(PacketV1) }

type testDataOptFunc func(*testData)

func TestPacketV1(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(...testDataOptFunc) testFn
		testParamsOutFn func(*testing.T) []testDataOptFunc
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(testDataOpts ...testDataOptFunc) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var d testData
				for _, testDataOpt := range testDataOpts {
					testDataOpt(&d)
				}

				input, output, xErr := d.rawPacketV1(), d.rawBytes, d.err
				if d.altPacket != nil {
					input = d.altPacketV1()
				}

				var have = new(bytes.Buffer)
				n, err := have.ReadFrom(&input)
				assert.Equal(t, int64(len(output)), n)
				assert.Equal(t, xErr, err)

				assert.Equal(t, output, have.Bytes())
			}
		},
		"Write": func(testDataOpts ...testDataOptFunc) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var d testData
				for _, testDataOpt := range testDataOpts {
					testDataOpt(&d)
				}

				input, output, xErr := d.rawBytes, d.rawPacketV1(), d.err

				var packet PacketV1
				n, err := (&packet).Write(input)

				assert.Equal(t, len(input), n, "the data length")
				assert.ErrorIs(t, err, xErr)

				assert.Equal(t, output.Type, packet.Type, "the packet type")
				assert.Equal(t, output.Namespace, packet.Namespace, "the namespace:")
				assert.Equal(t, output.AckID, packet.AckID, "the ackID")
			}
		},
		"WriteTo": func(testDataOpts ...testDataOptFunc) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var d testData
				for _, testDataOpt := range testDataOpts {
					testDataOpt(&d)
				}

				input, output, xErr := d.rawPacketV1(), d.rawBytes, d.err
				if d.altPacket != nil {
					input = d.altPacketV1()
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
		"CONNECT": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`0`)
			asPacket := *NewPacketV1().WithNamespace("/").(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"DISCONNECT": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`1/admin`)
			asPacket := *NewPacketV1().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`2["hello",1]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT with AckID": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`2/admin,456["project:delete",123]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"ACK": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`3/admin,456[]`)
			asPacket := *NewPacketV1().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"ERROR": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`4/admin,"Not authorized"`)
			asPacket := *NewPacketV1().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`0/admin`)
			asPacket := *NewPacketV1().
				WithNamespace("/admin").(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"CONNECT /admin ns and extra info": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`0/admin?token=1234&uid=abcd`)
			asPacket := *NewPacketV1().
				WithNamespace("/admin").(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) {
					as2Packet := asPacket
					as2Packet.WithNamespace("/admin?token=1234&uid=abcd")
					d.altPacket = as2Packet
				},
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT with Binary": func(t2 *testing.T) []testDataOptFunc {
			opts = append(opts, itst.DoNotTest("Write"))

			asError := ErrBinaryDataUnsupported
			asBytes := []byte(`2[]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{strings.NewReader("binary data")}).(*PacketV1)

			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = asError },
			}
		},
		"EVENT with Binary unsupported": func(*testing.T) []testDataOptFunc {
			asBytes := []byte(`2["unsupported",{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}]`)
			asPacket := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{
					"unsupported",
					map[string]interface{}{"base64": true, "data": "xAtiaW5hcnkgZGF0YQ=="},
				}).(*PacketV1)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT with Object": func(*testing.T) []testDataOptFunc {
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
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawBytes = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)...))
		}
	}
}
