package transport

import (
	"reflect"
	"testing"
	"time"

	"net/http/httptest"
	"strings"
	"sync"

	eiop "github.com/njones/socketio/engineio/protocol"
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

		tr := NewPollingTransport(10*time.Millisecond)(SessionID("12345"), test.codec)

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
			if err != nil {
				t.Error(err)
			}
			q.Wait()

			have := <-h
			want := test.packet

			if !reflect.DeepEqual(have, want) {
				t2.Errorf("have: %v want: %v", have, want)
			}
		})

		// Send Test
		t.Run(test.name, func(t2 *testing.T) {
			tr.Send(test.packet)

			r := httptest.NewRequest("GET", "http://example.com", nil)
			w := httptest.NewRecorder()

			err := tr.Run(w, r)
			if err != nil {
				t.Error(err)
			}

			have := w.Body.String()
			want := test.data

			if have != want {
				t2.Errorf("have: %q want: %q", have, want)
			}
		})
	}
}
