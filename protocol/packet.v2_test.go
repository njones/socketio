package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketV2Read(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(pac PacketV2, want []byte, xerr error) testFn
		testParamsOutFn func(*testing.T) (pac PacketV2, want []byte, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(pac PacketV2, want []byte, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := have.ReadFrom(&pac)
				assert.Equal(t, int64(len(want)), n)
				assert.Equal(t, xerr, err)

				assert.Equal(t, want, have.Bytes())
			}
		},
		"WriteTo": func(pac PacketV2, want []byte, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := (&pac).WriteTo(have)
				assert.Equal(t, int64(len(want)), n)
				assert.Equal(t, xerr, err)

				assert.Equal(t, want, have.Bytes())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`0`)
			data := *NewPacketV2().(*PacketV2)
			return data, want, nil
		},
		"DISCONNECT": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`1/admin`)
			data := *NewPacketV2().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV2)
			return data, want, nil
		},
		"EVENT": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`2["hello",1]`)
			data := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"hello", 1.0}).(*PacketV2)
			return data, want, nil
		},
		"EVENT with AckID": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`2/admin,456["project:delete",123]`)
			data := *NewPacketV2().
				WithNamespace("/admin").
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ACK": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`3/admin,456[]`)
			data := *NewPacketV2().
				WithNamespace("/admin").
				WithType(AckPacket.Byte()).
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ERROR": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`4/admin,"Not authorized"`)
			data := *NewPacketV2().
				WithNamespace("/admin").
				WithType(ErrorPacket.Byte()).
				WithData(&notAuthorized).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`51-["hello",{"base64":true,"data":"xAMBAgM="}]`)
			data := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT with AckID": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`51-/admin,456["hello",{"base64":true,"data":"xAMBAgM="}]`)
			data := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`0/admin`)
			data := *NewPacketV2().
				WithNamespace("/admin").(*PacketV2)
			return data, want, nil
		},
		"CONNECT /admin ns and extra info": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`0/admin?token=1234&uid=abcd`)
			data := *NewPacketV2().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV2)
			return data, want, nil
		},
		"EVENT with Binary": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`21-["name",{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}]`)
			data := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"name", strings.NewReader("binary data")}).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`3/site,444{"api-mocked.com":"member"}`)
			data := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": "member"}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object and Binary": func(*testing.T) (PacketV2, []byte, error) {
			want := []byte(`31-/site,444{"api-mocked.com":{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}}`)
			data := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": strings.NewReader("binary data")}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePacketV2(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(want []byte, pac PacketV2, xerr error) testFn
		testParamsOutFn func(*testing.T) (want []byte, pac PacketV2, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Write": func(data []byte, want PacketV2, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have PacketV2
				n, err := (&have).Write(data)

				assert.Equal(t, len(data), n, "the data length")
				assert.ErrorIs(t, err, xerr)

				assert.Equal(t, want.Type, have.Type, "the packet type")
				assert.Equal(t, want.Namespace, have.Namespace, "the namespace:")
				assert.Equal(t, want.AckID, have.AckID, "the ackID")
				assert.IsType(t, want.Data, have.Data, "the data (type)")

				switch _have := have.Data.(type) {
				case *packetDataArray:
					for i, wantx := range want.Data.(*packetDataArray).x {
						assert.Equal(t, wantx, _have.x[i])
					}
				}
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`0`)
			want := *NewPacketV2().WithNamespace("/").(*PacketV2)
			return data, want, nil
		},
		"DISCONNECT": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`1`)
			want := *NewPacketV2().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/").(*PacketV2)
			return data, want, nil
		},
		"EVENT": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`2["hello",1]`)
			want := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV2)
			return data, want, nil
		},
		"EVENT with AckID": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`2/admin,456["project:delete",123]`)
			want := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ACK": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`3/admin,456[]`)
			want := *NewPacketV2().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ERROR": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`4/admin,"Not authorized"`)
			want := *NewPacketV2().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(notAuthorized).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`51-["hello",{"base64":true,"data":"xAMBAgM="}]`)
			want := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT with AckID": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`51-/admin,456["hello",{"base64":true,"data":"xAMBAgM="}]`)
			want := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithAckID(456).
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`3/site,444{"api-mocked.com":"member"}`)
			want := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": "member"}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object and Binary": func(*testing.T) ([]byte, PacketV2, error) {
			data := []byte(`31-/site,444{"api-mocked.com":{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}}`)
			want := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": strings.NewReader("binary data")}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
