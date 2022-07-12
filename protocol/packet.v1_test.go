package protocol

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testoption func(*testing.T)

type testReadV1 func(PacketV1, []byte, error) func(*testing.T)
type testWriteV1 func([]byte, PacketV1, error) func(*testing.T)

var notAuthorized = "Not authorized"

func mergeReadV1(dst, src map[string]func() (PacketV1, []byte, error)) {
	for k, v := range src {
		dst[k] = v
	}
}

func mergeWriteV1(dst, src map[string]func() ([]byte, PacketV1, error)) {
	for k, v := range src {
		dst[k] = v
	}
}

func runTest(testnames ...string) testoption {
	return func(t *testing.T) {
		t.Helper()
		names := strings.SplitN(t.Name(), "/", 2)
		for _, testname := range testnames {
			if names[len(names)-1] == strings.ReplaceAll(testname, " ", "_") {
				return
			}
		}
		t.SkipNow()
	}
}

func skipTest(testnames ...string) testoption {
	return func(t *testing.T) {
		t.Helper()
		names := strings.SplitN(t.Name(), "/", 2)
		for _, testname := range testnames {
			if names[len(names)-1] == strings.ReplaceAll(testname, " ", "_") {
				t.SkipNow()
				return
			}
		}
	}
}

func TestPacketV1Read(t *testing.T) {
	var opts []testoption

	testchecks := map[string]func(checks ...testoption) testReadV1{
		".ReadFrom": func(checks ...testoption) testReadV1 {
			return func(pac PacketV1, want []byte, xerr error) func(*testing.T) {
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
		".WriteTo": func(checks ...testoption) testReadV1 {
			return func(pac PacketV1, want []byte, xerr error) func(*testing.T) {
				return func(t *testing.T) {
					for _, check := range checks {
						check(t)
					}

					var have = new(bytes.Buffer)
					n, err := pac.WriteTo(have)
					assert.Equal(t, int64(len(want)), n)
					assert.Equal(t, xerr, err)

					assert.Equal(t, want, have.Bytes())
				}
			}
		},
	}

	spec := map[string]func() (PacketV1, []byte, error){
		"CONNECT": func() (PacketV1, []byte, error) {
			want := []byte(`0`)
			data := *NewPacketV1().(*PacketV1)
			return data, want, nil
		},
		"DISCONNECT": func() (PacketV1, []byte, error) {
			want := []byte(`1/admin`)
			data := *NewPacketV1().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV1)
			return data, want, nil
		},
		"EVENT": func() (PacketV1, []byte, error) {
			want := []byte(`2["hello",1]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"hello", 1.0}).(*PacketV1)
			return data, want, nil
		},
		"EVENT with AckID": func() (PacketV1, []byte, error) {
			want := []byte(`2/admin,456["project:delete",123]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ACK": func() (PacketV1, []byte, error) {
			want := []byte(`3/admin,456[]`)
			data := *NewPacketV1().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ERROR": func() (PacketV1, []byte, error) {
			want := []byte(`4/admin,"Not authorized"`)
			data := *NewPacketV1().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV1)
			return data, want, nil
		},
	}

	extra := map[string]func() (PacketV1, []byte, error){
		"CONNECT /admin ns": func() (PacketV1, []byte, error) {
			want := []byte(`0/admin`)
			data := *NewPacketV1().
				WithNamespace("/admin").(*PacketV1)
			return data, want, nil
		},
		"CONNECT /admin ns and extra info": func() (PacketV1, []byte, error) {
			want := []byte(`0/admin?token=1234&uid=abcd`)
			data := *NewPacketV1().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV1)
			return data, want, nil
		},
		"EVENT with Binary": func() (PacketV1, []byte, error) {
			want := []byte(`2[]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{strings.NewReader("binary data")}).(*PacketV1)
			return data, want, ErrBinaryDataUnsupported
		},
		"EVENT with Object": func() (PacketV1, []byte, error) {
			want := []byte(`2["hello",{"playground":"world","wake":{"won":["too",3]}}]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"hello", map[string]interface{}{
					"playground": "world",
					"wake": map[string]interface{}{
						"won": []interface{}{"too", 3},
					},
				}}).(*PacketV1)
			return data, want, nil
		},
	}

	mergeReadV1(extra, spec)
	for name, testing := range extra {
		for fill, testcheck := range testchecks {
			t.Run(name+fill, testcheck(opts...)(testing()))
		}
	}
}

func TestWritePacketV1(t *testing.T) {
	var opts []testoption

	testcheck := func(checks ...testoption) testWriteV1 {
		return func(data []byte, want PacketV1, xerr error) func(*testing.T) {
			return func(t *testing.T) {
				for _, check := range checks {
					check(t)
				}

				var pac PacketV1
				n, err := (&pac).Write(data)

				assert.Equal(t, len(data), n, "the data length")
				assert.ErrorIs(t, err, xerr)

				assert.Equal(t, want.Type, pac.Type, "the packet type")
				assert.Equal(t, want.Namespace, pac.Namespace, "the namespace:")
				assert.Equal(t, want.AckID, pac.AckID, "the ackID")
			}
		}
	}

	spec := map[string]func() ([]byte, PacketV1, error){
		"CONNECT": func() ([]byte, PacketV1, error) {
			data := []byte(`0`)
			want := *NewPacketV1().WithNamespace("/").(*PacketV1)
			return data, want, nil
		},
		"DISCONNECT": func() ([]byte, PacketV1, error) {
			data := []byte(`1`)
			want := *NewPacketV1().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/").(*PacketV1)
			return data, want, nil
		},
		"EVENT": func() ([]byte, PacketV1, error) {
			data := []byte(`2["hello",1]`)
			want := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV1)
			return data,
				want, nil
		},
		"EVENT with AckID": func() ([]byte, PacketV1, error) {
			data := []byte(`2/admin,456["project:delete",123]`)
			want := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ACK": func() ([]byte, PacketV1, error) {
			data := []byte(`3/admin,456[]`)
			want := *NewPacketV1().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ERROR": func() ([]byte, PacketV1, error) {
			data := []byte(`4/admin,"Not authorized"`)
			want := *NewPacketV1().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(notAuthorized).(*PacketV1)
			return data, want, nil
		},
	}

	extra := map[string]func() ([]byte, PacketV1, error){
		"EVENT with Binary": func() ([]byte, PacketV1, error) {
			data := []byte(`2["unknown", {"base64":true,"data":"xAtiaW5hcnkgZGF0YQ=="}]`)
			want := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{
					"unknown",
					map[string]interface{}{"base64": true, "data": "xAtiaW5hcnkgZGF0YQ=="},
				}).(*PacketV1)
			return data, want, nil
		},
	}

	mergeWriteV1(extra, spec)

	for name, testing := range extra {
		t.Run(name, testcheck(opts...)(testing()))
	}
}
