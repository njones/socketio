package protocol

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	itst "github.com/njones/socketio/internal/test"
	"github.com/stretchr/testify/assert"
)

func (d testData) rawPacketV3() PacketV3 { return d.rawPacket.(PacketV3) }
func (d testData) altPacketV3() PacketV3 { return (d.altPacket).(PacketV3) }

func TestPacketV3(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(...testDataOptFunc) testFn
		testParamsOutFn func(*testing.T) []testDataOptFunc
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(testDataOpts ...testDataOptFunc) testFn { //func(input PacketV3, output [][]byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var d testData
				for _, testDataOpt := range testDataOpts {
					testDataOpt(&d)
				}

				input, output, xErr := d.rawPacketV3(), d.rawByteSlice, d.err
				if d.altPacket != nil {
					input = d.altPacketV3()
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
		"Write": func(testDataOpts ...testDataOptFunc) testFn { // func(output PacketV3, input [][]byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var d testData
				for _, testDataOpt := range testDataOpts {
					testDataOpt(&d)
				}

				input, output, xErr := d.rawByteSlice, d.rawPacketV3(), d.err

				var have PacketV3
				n, err := (&have).Write(input[0])

				assert.Equal(t, len(input[0]), n, "the data length")
				assert.ErrorIs(t, err, xErr)

				go func() {
					for i, next := range have.incoming {
						err := next(bytes.NewReader(input[i+1]))
						assert.NoError(t, err)
					}
				}()

				assert.Equal(t, output.Type, have.Type, "the packet type")
				assert.Equal(t, output.Namespace, have.Namespace, "the namespace:")
				assert.Equal(t, output.AckID, have.AckID, "the ackID")
				assert.IsType(t, output.Data, have.Data, "the data (type)")

				switch _have := have.Data.(type) {
				case *packetDataArray:
					for i, wantx := range output.Data.(*packetDataArray).x {
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
		"WriteTo": func(testDataOpts ...testDataOptFunc) testFn { //func(input PacketV3, output [][]byte, xErr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var d testData
				for _, testDataOpt := range testDataOpts {
					testDataOpt(&d)
				}

				input, output, xErr := d.rawPacketV3(), d.rawByteSlice, d.err
				if d.altPacket != nil {
					input = d.altPacketV3()
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
		"CONNECT": func(*testing.T) []testDataOptFunc {
			var (
				asPacket = *NewPacketV3().WithNamespace("/").(*PacketV3)
				asBytes  = [][]byte{[]byte(`0`)}
			)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"DISCONNECT": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`1/admin`)}
			asPacket := *NewPacketV3().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`2["hello",1]`)}
			asPacket := *NewPacketV3().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT with AckID": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`2/admin,456["project:delete",123]`)}
			asPacket := *NewPacketV3().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"ACK": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`3/admin,456[]`)}
			asPacket := *NewPacketV3().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"ERROR": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`4/admin,"Not authorized"`)}
			asPacket := *NewPacketV3().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"BINARY EVENT": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{
				[]byte(`51-["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			asPacket := *NewPacketV3().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"BINARY EVENT with AckID": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{
				[]byte(`51-/admin,456["hello",{"_placeholder":true,"num":0}]`),
				{0x01, 0x02, 0x03},
			}
			asPacket := *NewPacketV3().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).
				WithAckID(456).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`0/admin`)}
			asPacket := *NewPacketV3().
				WithNamespace("/admin").(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"CONNECT /admin ns and extra info": func(*testing.T) []testDataOptFunc {
			asBytes := [][]byte{[]byte(`0/admin?token=1234&uid=abcd`)}
			asPacket := *NewPacketV3().
				WithNamespace("/admin").(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) {
					as2Packet := asPacket
					as2Packet.WithNamespace("/admin?token=1234&uid=abcd")
					d.altPacket = as2Packet
				},
				func(d *testData) { d.rawByteSlice = asBytes },
				func(d *testData) { d.err = nil },
			}
		},
		"EVENT with Binary": func(*testing.T) []testDataOptFunc {
			opts = append(opts, itst.DoNotTest("Write"))

			asBytes := [][]byte{[]byte(`21-[{"_placeholder":true,"num":0}]`)}
			asPacket := *NewPacketV3().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{strings.NewReader("binary data")}).(*PacketV3)
			return []testDataOptFunc{
				func(d *testData) { d.rawPacket = asPacket },
				func(d *testData) { d.rawByteSlice = asBytes },
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
