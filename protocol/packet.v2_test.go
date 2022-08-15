package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testReadV2 func(PacketV2, []byte, error) func(*testing.T)
type testWriteV2 func([]byte, PacketV2, error) func(*testing.T)

func mergeReadV2(dst, src map[string]func() (PacketV2, []byte, error)) {
	for k, v := range src {
		dst[k] = v
	}
}

func mergeWriteV2(dst, src map[string]func() ([]byte, PacketV2, error)) {
	for k, v := range src {
		dst[k] = v
	}
}

func TestPacketV2Read(t *testing.T) {
	var opts = []testoption{}

	testchecks := map[string]func(checks ...testoption) testReadV2{
		".ReadFrom": func(checks ...testoption) testReadV2 {
			return func(pac PacketV2, want []byte, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, check := range checks {
						check(t)
					}

					var have = new(bytes.Buffer)
					n, err := have.ReadFrom(&pac)
					assert.Equal(t, int64(len(want)), n)
					assert.Equal(t, xerr, err)

					assert.Equal(t, want, have.Bytes())
				}
			}
		},
		".WriteTo": func(checks ...testoption) testReadV2 {
			return func(pac PacketV2, want []byte, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, check := range checks {
						check(t)
					}

					var have = new(bytes.Buffer)
					n, err := (&pac).WriteTo(have)
					assert.Equal(t, int64(len(want)), n)
					assert.Equal(t, xerr, err)

					assert.Equal(t, want, have.Bytes())
				}
			}
		},
	}

	spec := map[string]func() (PacketV2, []byte, error){
		"CONNECT": func() (PacketV2, []byte, error) {
			want := []byte(`0`)
			data := *NewPacketV2().(*PacketV2)
			return data, want, nil
		},
		"DISCONNECT": func() (PacketV2, []byte, error) {
			want := []byte(`1/admin`)
			data := *NewPacketV2().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV2)
			return data, want, nil
		},
		"EVENT": func() (PacketV2, []byte, error) {
			want := []byte(`2["hello",1]`)
			data := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"hello", 1.0}).(*PacketV2)
			return data, want, nil
		},
		"EVENT with AckID": func() (PacketV2, []byte, error) {
			want := []byte(`2/admin,456["project:delete",123]`)
			data := *NewPacketV2().
				WithNamespace("/admin").
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ACK": func() (PacketV2, []byte, error) {
			want := []byte(`3/admin,456[]`)
			data := *NewPacketV2().
				WithNamespace("/admin").
				WithType(AckPacket.Byte()).
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ERROR": func() (PacketV2, []byte, error) {
			want := []byte(`4/admin,"Not authorized"`)
			data := *NewPacketV2().
				WithNamespace("/admin").
				WithType(ErrorPacket.Byte()).
				WithData(&notAuthorized).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT": func() (PacketV2, []byte, error) {
			want := []byte(`51-["hello",{"base64":true,"data":"xAMBAgM="}]`)
			data := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT with AckID": func() (PacketV2, []byte, error) {
			want := []byte(`51-/admin,456["hello",{"base64":true,"data":"xAMBAgM="}]`)
			data := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
	}

	extra := map[string]func() (PacketV2, []byte, error){
		"CONNECT /admin ns": func() (PacketV2, []byte, error) {
			want := []byte(`0/admin`)
			data := *NewPacketV2().
				WithNamespace("/admin").(*PacketV2)
			return data, want, nil
		},
		"CONNECT /admin ns and extra info": func() (PacketV2, []byte, error) {
			want := []byte(`0/admin?token=1234&uid=abcd`)
			data := *NewPacketV2().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV2)
			return data, want, nil
		},
		"EVENT with Binary": func() (PacketV2, []byte, error) {
			want := []byte(`21-["name",{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}]`)
			data := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"name", strings.NewReader("binary data")}).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object": func() (PacketV2, []byte, error) {
			want := []byte(`3/site,444{"api-mocked.com":"member"}`)
			data := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": "member"}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object and Binary": func() (PacketV2, []byte, error) {
			want := []byte(`31-/site,444{"api-mocked.com":{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}}`)
			data := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": strings.NewReader("binary data")}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
	}

	mergeReadV2(extra, spec)
	for name, testing := range extra {
		for fill, testcheck := range testchecks {
			t.Run(name+fill, testcheck(opts...)(testing()))
		}
	}
}

func TestWritePacketV2(t *testing.T) {
	var opts = []testoption{}

	testcheck := func(checks ...testoption) testWriteV2 {
		return func(data []byte, want PacketV2, xerr error) func(*testing.T) {
			return func(t *testing.T) {
				for _, check := range checks {
					check(t)
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
		}
	}

	spec := map[string]func() ([]byte, PacketV2, error){
		"CONNECT": func() ([]byte, PacketV2, error) {
			data := []byte(`0`)
			want := *NewPacketV2().WithNamespace("/").(*PacketV2)
			return data, want, nil
		},
		"DISCONNECT": func() ([]byte, PacketV2, error) {
			data := []byte(`1`)
			want := *NewPacketV2().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/").(*PacketV2)
			return data, want, nil
		},
		"EVENT": func() ([]byte, PacketV2, error) {
			data := []byte(`2["hello",1]`)
			want := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV2)
			return data, want, nil
		},
		"EVENT with AckID": func() ([]byte, PacketV2, error) {
			data := []byte(`2/admin,456["project:delete",123]`)
			want := *NewPacketV2().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ACK": func() ([]byte, PacketV2, error) {
			data := []byte(`3/admin,456[]`)
			want := *NewPacketV2().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV2)
			return data, want, nil
		},
		"ERROR": func() ([]byte, PacketV2, error) {
			data := []byte(`4/admin,"Not authorized"`)
			want := *NewPacketV2().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(notAuthorized).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT": func() ([]byte, PacketV2, error) {
			data := []byte(`51-["hello",{"base64":true,"data":"xAMBAgM="}]`)
			want := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return data, want, nil
		},
		"BINARY EVENT with AckID": func() ([]byte, PacketV2, error) {
			data := []byte(`51-/admin,456["hello",{"base64":true,"data":"xAMBAgM="}]`)
			want := *NewPacketV2().
				WithType(BinaryEventPacket.Byte()).
				WithNamespace("/admin").
				WithAckID(456).
				WithData([]interface{}{"hello", bytes.NewReader([]byte{0x01, 0x02, 0x03})}).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object": func() ([]byte, PacketV2, error) {
			data := []byte(`3/site,444{"api-mocked.com":"member"}`)
			want := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": "member"}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
		"ACK with Object and Binary": func() ([]byte, PacketV2, error) {
			data := []byte(`31-/site,444{"api-mocked.com":{"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}}`)
			want := *NewPacketV2().
				WithNamespace("/site").
				WithType(AckPacket.Byte()).
				WithData(map[string]interface{}{"api-mocked.com": strings.NewReader("binary data")}).
				WithAckID(444).(*PacketV2)
			return data, want, nil
		},
	}

	extra := map[string]func() ([]byte, PacketV2, error){}

	mergeWriteV2(extra, spec)

	for name, testing := range extra {
		t.Run(name, testcheck(opts...)(testing()))
	}
}
