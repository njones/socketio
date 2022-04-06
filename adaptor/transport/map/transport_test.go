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
	"github.com/njones/socketio/transport"
	"github.com/stretchr/testify/assert"
)

func TestMapTransportSend(t *testing.T) {
	sess.GenerateID = func() tmap.SocketID { return tmap.SocketID("ABC123") }

	c := eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	wgrp := new(sync.WaitGroup)

	etr := eiot.NewPollingTransport(10*time.Millisecond)(tmap.SessionID("12345"), c)
	str := tmap.NewMapTransport(siop.NewPacketV2)

	sid, err := str.Add(etr)
	if err != nil {
		t.Fatal(err)
	}

	errChan := make(chan error, 1)
	rec := httptest.NewRecorder()

	wgrp.Add(1)
	go func() {
		defer wgrp.Done()
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- etr.Run(rec, req)
	}()

	str.Send(sid, "This is cool")
	wgrp.Wait()

	if err := <-errChan; err != nil {
		t.Fatal(err)
	}

	have := rec.Body.String()
	want := thisIsCoolText

	assert.Equal(t, want, have)
}

var thisIsCoolText = `16:40"This is cool"`

func TestMapTransportReceive(t *testing.T) {
	sess.GenerateID = func() tmap.SocketID { return tmap.SocketID("ABC123") }

	c := eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	wgrp := new(sync.WaitGroup)

	etr := eiot.NewPollingTransport(10*time.Millisecond)(tmap.SessionID("12345"), c)
	str := tmap.NewMapTransport(siop.NewPacketV2)

	sid, err := str.Add(etr)
	if err != nil {
		t.Fatal(err)
	}

	errChan := make(chan error, 1)
	rec := httptest.NewRecorder()

	wgrp.Add(1)
	go func() {
		defer wgrp.Done()

		req, err := http.NewRequest("POST", "/", strings.NewReader(thisIsCoolText))
		if err != nil {
			errChan <- err
			return
		}

		errChan <- etr.Run(rec, req)
	}()
	wgrp.Wait()

	if err := <-errChan; err != nil {
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

	assert.Equal(t, want, have)
}
