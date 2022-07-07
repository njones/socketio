package transport

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	"github.com/stretchr/testify/assert"
)

func TestPollingTransportReceive(t *testing.T) {
	cV2 := Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	var tests = []struct {
		name   string
		data   string
		packet eiop.Packet
		codec  Codec
	}{
		{
			name:   "[message]",
			data:   "6:4Hello",
			packet: eiop.Packet{T: eiop.MessagePacket, D: "Hello"},
			codec:  cV2,
		},
	}

	for _, test := range tests {

		tr := NewPollingTransport(1000, 10*time.Millisecond)(SessionID("12345"), test.codec)

		// Receive Test
		t.Run(test.name, func(t2 *testing.T) {
			q := new(sync.WaitGroup)
			h := make(chan eiop.Packet, 1)

			q.Add(1)
			go func() {
				defer q.Done()
				h <- <-tr.Receive()
			}()

			r := httptest.NewRequest("POST", "http://example.com", strings.NewReader(test.data))
			w := httptest.NewRecorder()

			err := tr.Run(w, r)
			assert.NoError(t, err)
			q.Wait()

			have := <-h
			want := test.packet

			assert.Equal(t, want, have)
		})

		// Send Test
		t.Run(test.name, func(t2 *testing.T) {
			tr.Send(test.packet)

			r := httptest.NewRequest("GET", "http://example.com", nil)
			w := httptest.NewRecorder()

			err := tr.Run(w, r)
			assert.NoError(t, err)

			have := w.Body.String()
			want := test.data

			assert.Equal(t, want, have)
		})
	}
}
