package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	eiop "github.com/njones/socketio/engineio/protocol"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketTransport(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func([]eiop.Packet, []string, Codec) testFn
		testParamsOutFn func(*testing.T) ([]eiop.Packet, []string, Codec)
	)

	runWithOptions := map[string]testParamsInFn{
		"Send": func(packets []eiop.Packet, messages []string, codec Codec) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				tr := NewWebsocketTransport(1000)(SessionID("12345"), codec)

				wai := new(sync.WaitGroup)
				wai.Add(1)
				server := httptest.NewServer(testRunHandler{wait: wai, t: t, fn: tr.Run})

				wsURL := strings.ReplaceAll(server.URL, "http", "ws")
				wsConn, _, _, err := ws.Dial(context.TODO(), wsURL+"/engine.io")
				wai.Done()

				assert.NoError(t, err)

				ping, err := wsutil.ReadServerText(wsConn)
				assert.NoError(t, err)

				err = wsutil.WriteClientText(wsConn, append([]byte{'3'}, ping[1:]...))
				assert.NoError(t, err)

				for _, packet := range packets {
					tr.Send(packet)
				}

				var x int
				for _, message := range messages {
					data, err := wsutil.ReadServerText(wsConn)
					assert.NoError(t, err)

					assert.Equal(t, message, string(data))
					x++
				}

				assert.Equal(t, len(messages), x)
			}
		},
		"Receive": func(packets []eiop.Packet, messages []string, codec Codec) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				tr := NewWebsocketTransport(1000)(SessionID("12345"), codec)

				wai := new(sync.WaitGroup)
				wai.Add(1)
				server := httptest.NewServer(testRunHandler{wait: wai, t: t, fn: tr.Run})

				wsURL := strings.ReplaceAll(server.URL, "http", "ws")
				wsConn, _, _, err := ws.Dial(context.TODO(), wsURL+"/engine.io")
				wai.Done()

				assert.NoError(t, err)

				ping, err := wsutil.ReadServerText(wsConn)
				assert.NoError(t, err)

				err = wsutil.WriteClientText(wsConn, append([]byte{'3'}, ping[1:]...))
				assert.NoError(t, err)

				for _, msg := range messages {
					err = wsutil.WriteClientText(wsConn, []byte(msg))
					assert.NoError(t, err)
				}

				var n int
				receive := tr.Receive()
				for have := range receive {
					want := packets[n]
					if want.T != eiop.NoopPacket {
						assert.Equal(t, want, have)
					}
					n++
					if n == len(packets) {
						assert.Equal(t, 0, len(receive))
						break
					}
				}

			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Version 2 Single": func(*testing.T) (packets []eiop.Packet, message []string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}}, []string{"4Hello"}, cV2
		},
		"Version 2 Payload": func(*testing.T) (packets []eiop.Packet, message []string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}, {T: eiop.MessagePacket, D: "World"}}, []string{"4Hello", "4World"}, cV2
		},
		"Version 3 Single": func(*testing.T) (packets []eiop.Packet, message []string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV3,
				PacketDecoder:  eiop.NewPacketDecoderV3,
				PayloadEncoder: eiop.NewPayloadEncoderV3,
				PayloadDecoder: eiop.NewPayloadDecoderV3,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}}, []string{"4Hello"}, cV2
		},
		"Version 4 Single": func(*testing.T) (packets []eiop.Packet, message []string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV4,
				PacketDecoder:  eiop.NewPacketDecoderV4,
				PayloadEncoder: eiop.NewPayloadEncoderV4,
				PayloadDecoder: eiop.NewPayloadDecoderV4,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}}, []string{"4Hello"}, cV2
		},
		"Version 4 Payload": func(*testing.T) (packets []eiop.Packet, message []string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV4,
				PacketDecoder:  eiop.NewPacketDecoderV4,
				PayloadEncoder: eiop.NewPayloadEncoderV4,
				PayloadDecoder: eiop.NewPayloadDecoderV4,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}, {T: eiop.MessagePacket, D: "World"}}, []string{"4Hello", "4World"}, cV2
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

type testRunHandler struct {
	fn   func(http.ResponseWriter, *http.Request, ...Option) error
	t    *testing.T
	opts []Option
	wait *sync.WaitGroup
}

func (h testRunHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = context.WithValue(ctx, serverSetupComplete, h.wait)
	err := h.fn(w, r.WithContext(ctx), h.opts...)
	assert.NoError(h.t, err)
}
