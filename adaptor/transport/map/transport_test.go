package tmap_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
	siop "github.com/njones/socketio/protocol"
	sess "github.com/njones/socketio/session"
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
	t.Error(sid, err)

	rec := httptest.NewRecorder()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest("GET", "/", nil)
		err := etr.Run(rec, req)
		t.Log(err)
	}()

	str.Send(sid, "This is cool")
	wg.Wait()

	t.Error(rec.Body.String())
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
	t.Error(sid, err)

	rec := httptest.NewRecorder()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest("POST", "/", strings.NewReader(`16:40"This is cool"`))
		err := etr.Run(rec, req)
		t.Log(err)
	}()
	wg.Wait()

	k := <-str.Receive(sid)
	t.Errorf("%#v %#v", k, *k.Data.(*string))
}
