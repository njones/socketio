package tmap_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
	siop "github.com/njones/socketio/protocol"
	sess "github.com/njones/socketio/session"
	"github.com/njones/socketio/transport"
)

func TestMapTransport(t *testing.T) {
	c := eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	etr := eiot.NewPollingTransport(10*time.Millisecond)(tmap.SessionID("12345"), c)
	str := tmap.NewMapTransport(siop.NewPacketV3)

	sess.GenerateID = func() tmap.SocketID {
		return tmap.SocketID("ABC123")
	}

	sid, err := str.Add(etr)
	if err != nil {
		t.Fatal(err)
	}

	errch := make(chan error, 1)
	rec := httptest.NewRecorder()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest("GET", "/", nil)
		errch <- etr.Run(rec, req)
	}()

	str.Send(sid, "This is cool")
	wg.Wait()

	if err := <-errch; err != nil {
		t.Fatal(err)
	}

	have := rec.Body.String()
	want := `16:40"This is cool"`

	if have != want {
		t.Fatalf("have: %q want: %q", have, want)
	}
}

func TestMapTransportBackwards(t *testing.T) {
	c := eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	etr := eiot.NewPollingTransport(10*time.Millisecond)(tmap.SessionID("12345"), c)
	str := tmap.NewMapTransport(siop.NewPacketV3)

	sess.GenerateID = func() tmap.SocketID {
		return tmap.SocketID("ABC123")
	}

	sid, err := str.Add(etr)
	if err != nil {
		t.Fatal(err)
	}

	errch := make(chan error, 1)
	rec := httptest.NewRecorder()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest("POST", "/", strings.NewReader(`16:40"This is cool"`))
		errch <- etr.Run(rec, req)
	}()
	wg.Wait()

	if err := <-errch; err != nil {
		t.Fatal(err)
	}

	data := "This is cool"

	have := <-str.Receive(sid)
	want := transport.Socket{
		Type:      byte(eiop.OpenPacket),
		Namespace: "/",
		AckID:     0x0,
		Data:      &data,
	}

	if !reflect.DeepEqual(have, want) {
		t.Fatalf("have: %#v want: %#v", have, want)
	}
}
