package protocol

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testOption func(*testing.T)

var notAuthorized = "Not authorized"

var testingName = strings.NewReplacer(" ", "_")

func runTest(testNames ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			if testName == "" || testName == "*" {
				return
			}

			want := testingName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				return
			}
		}
		t.SkipNow()
	}
}

func skipTest(testNames ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			want := testingName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				t.SkipNow()
			}
		}
	}
}

func TestPacketV1Read(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(pac PacketV1, want []byte, xerr error) testFn
		testParamsOutFn func(*testing.T) (pac PacketV1, want []byte, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"ReadFrom": func(pac PacketV1, want []byte, xerr error) testFn {
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
		"WriteTo": func(pac PacketV1, want []byte, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var have = new(bytes.Buffer)
				n, err := pac.WriteTo(have)
				assert.Equal(t, int64(len(want)), n)
				assert.Equal(t, xerr, err)

				assert.Equal(t, want, have.Bytes())
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`0`)
			data := *NewPacketV1().(*PacketV1)
			return data, want, nil
		},
		"DISCONNECT": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`1/admin`)
			data := *NewPacketV1().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/admin").(*PacketV1)
			return data, want, nil
		},
		"EVENT": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`2["hello",1]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithData([]interface{}{"hello", 1.0}).(*PacketV1)
			return data, want, nil
		},
		"EVENT with AckID": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`2/admin,456["project:delete",123]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ACK": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`3/admin,456[]`)
			data := *NewPacketV1().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ERROR": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`4/admin,"Not authorized"`)
			data := *NewPacketV1().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(&notAuthorized).(*PacketV1)
			return data, want, nil
		},

		// extra
		"CONNECT /admin ns": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`0/admin`)
			data := *NewPacketV1().
				WithNamespace("/admin").(*PacketV1)
			return data, want, nil
		},
		"CONNECT /admin ns and extra info": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`0/admin?token=1234&uid=abcd`)
			data := *NewPacketV1().
				WithNamespace("/admin?token=1234&uid=abcd").(*PacketV1)
			return data, want, nil
		},
		"EVENT with Binary": func(*testing.T) (PacketV1, []byte, error) {
			want := []byte(`2[]`)
			data := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{strings.NewReader("binary data")}).(*PacketV1)
			return data, want, ErrBinaryDataUnsupported
		},
		"EVENT with Object": func(*testing.T) (PacketV1, []byte, error) {
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

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestWritePacketV1(t *testing.T) {
	var opts = []func(*testing.T){runTest("")}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(want []byte, pac PacketV1, xerr error) testFn
		testParamsOutFn func(*testing.T) (want []byte, pac PacketV1, xerr error)
	)

	runWithOptions := map[string]testParamsInFn{
		"Write": func(data []byte, want PacketV1, xerr error) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				var pac PacketV1
				n, err := (&pac).Write(data)

				assert.Equal(t, len(data), n, "the data length")
				assert.ErrorIs(t, err, xerr)

				assert.Equal(t, want.Type, pac.Type, "the packet type")
				assert.Equal(t, want.Namespace, pac.Namespace, "the namespace:")
				assert.Equal(t, want.AckID, pac.AckID, "the ackID")
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"CONNECT": func(*testing.T) ([]byte, PacketV1, error) {
			data := []byte(`0`)
			want := *NewPacketV1().WithNamespace("/").(*PacketV1)
			return data, want, nil
		},
		"DISCONNECT": func(*testing.T) ([]byte, PacketV1, error) {
			data := []byte(`1`)
			want := *NewPacketV1().
				WithType(DisconnectPacket.Byte()).
				WithNamespace("/").(*PacketV1)
			return data, want, nil
		},
		"EVENT": func(*testing.T) ([]byte, PacketV1, error) {
			data := []byte(`2["hello",1]`)
			want := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/").
				WithData([]interface{}{"hello", 1.0}).(*PacketV1)
			return data,
				want, nil
		},
		"EVENT with AckID": func(*testing.T) ([]byte, PacketV1, error) {
			data := []byte(`2/admin,456["project:delete",123]`)
			want := *NewPacketV1().
				WithType(EventPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{"project:delete", 123.0}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ACK": func(*testing.T) ([]byte, PacketV1, error) {
			data := []byte(`3/admin,456[]`)
			want := *NewPacketV1().
				WithType(AckPacket.Byte()).
				WithNamespace("/admin").
				WithData([]interface{}{}).
				WithAckID(456).(*PacketV1)
			return data, want, nil
		},
		"ERROR": func(*testing.T) ([]byte, PacketV1, error) {
			data := []byte(`4/admin,"Not authorized"`)
			want := *NewPacketV1().
				WithType(ErrorPacket.Byte()).
				WithNamespace("/admin").
				WithData(notAuthorized).(*PacketV1)
			return data, want, nil
		},

		// extra
		"EVENT with Binary": func(*testing.T) ([]byte, PacketV1, error) {
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

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
