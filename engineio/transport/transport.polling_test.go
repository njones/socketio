package transport

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	itest "github.com/njones/socketio/internal/test"
	"github.com/stretchr/testify/assert"
)

var runTest = itest.RunTest

func TestPollingTransport(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func([]eiop.Packet, string, Codec) testFn
		testParamsOutFn func(*testing.T) ([]eiop.Packet, string, Codec)
	)

	runWithOptions := map[string]testParamsInFn{
		"Send": func(packets []eiop.Packet, message string, codec Codec) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				tr := NewPollingTransport(1000, 10*time.Millisecond)(SessionID("12345"), codec)

				for _, packet := range packets {
					tr.Send(packet)
				}

				r := httptest.NewRequest("GET", "http://example.com", nil)
				w := httptest.NewRecorder()

				err := tr.Run(w, r)
				assert.NoError(t, err)

				have := w.Body.String()
				want := message

				assert.Equal(t, want, have)
			}
		},
		"Receive": func(packets []eiop.Packet, message string, codec Codec) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				tr := NewPollingTransport(1000, 10*time.Millisecond)(SessionID("12345"), codec)

				r := httptest.NewRequest("POST", "http://example.com", strings.NewReader(message))
				w := httptest.NewRecorder()

				err := tr.Run(w, r)
				assert.NoError(t, err)

				var n int
				for have := range tr.Receive() {
					if len(packets) == n {
						// we send back a close socket internally
						assert.Equal(t, eiop.Packet{T: eiop.NoopPacket, D: socketClose{}}, have)
						break
					}
					want := packets[n]
					if want.T != eiop.NoopPacket {
						assert.Equal(t, want, have)
					}
					n++
				}
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"Version 2 Single": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}}, "6:4Hello", cV2
		},
		"Version 2 Payload": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}, {T: eiop.MessagePacket, D: "World"}}, "6:4Hello6:4World", cV2
		},
		"Version 3 Single": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV3,
				PacketDecoder:  eiop.NewPacketDecoderV3,
				PayloadEncoder: eiop.NewPayloadEncoderV3,
				PayloadDecoder: eiop.NewPayloadDecoderV3,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}}, "6:4Hello", cV2
		},
		"Version 4 Single": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV4,
				PacketDecoder:  eiop.NewPacketDecoderV4,
				PayloadEncoder: eiop.NewPayloadEncoderV4,
				PayloadDecoder: eiop.NewPayloadDecoderV4,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}}, "4Hello", cV2
		},
		"Version 4 Payload": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV4,
				PacketDecoder:  eiop.NewPacketDecoderV4,
				PayloadEncoder: eiop.NewPayloadEncoderV4,
				PayloadDecoder: eiop.NewPayloadDecoderV4,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello"}, {T: eiop.MessagePacket, D: "World"}}, "4Hello\x1e4World", cV2
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func TestPollingJSONPTransport(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func([]eiop.Packet, string, Codec) testFn
		testParamsOutFn func(*testing.T) ([]eiop.Packet, string, Codec)
	)

	runWithOptions := map[string]testParamsInFn{
		"JSONP": func(packets []eiop.Packet, message string, codec Codec) testFn {
			return func(t *testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				tr := NewPollingTransport(1000, 10*time.Millisecond)(SessionID("12345"), codec)

				for _, packet := range packets {
					tr.Send(packet)
				}

				r := httptest.NewRequest("GET", "http://example.com?j=10", nil)
				w := httptest.NewRecorder()

				err := tr.Run(w, r)
				assert.NoError(t, err)

				have := w.Body.String()
				want := message

				assert.Equal(t, want, have)
			}
		},
	}

	spec := map[string]testParamsOutFn{
		"No Quotes": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: "Hello World"}}, `___eio[10]("12:4Hello World");`, cV2
		},
		"With Quotes": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: `"Hello World"`}}, `___eio[10]("14:4\"Hello World\"");`, cV2
		},
		"With Quotes in Quotes": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: `""Hello World""`}}, `___eio[10]("16:4\"\"Hello World\"\"");`, cV2
		},
		"Binary": func(*testing.T) (packet []eiop.Packet, message string, codec Codec) {
			cV2 := Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}
			return []eiop.Packet{{T: eiop.MessagePacket, D: []byte{0x01, 0x02, 0x03, 0x04}}}, `___eio[10]("5:4\x01\x02\x03\x04");`, cV2
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}
