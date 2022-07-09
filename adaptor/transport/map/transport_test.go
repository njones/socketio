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
	siot "github.com/njones/socketio/transport"
	"github.com/stretchr/testify/assert"
)

func TestMapTransportSet(t *testing.T) {
	tsp := tmap.NewMapTransport(siop.NewPacketV2)
	id := "sio:aaa"
	a := siot.SocketID(id)

	nilSocketID := tsp.Receive(a)
	assert.Nil(t, nilSocketID)

	tspr := newMockTransporter(id)
	err := tsp.Set(a, tspr)
	assert.NoError(t, err)

	socketID := tsp.Receive(a)
	assert.NotNil(t, socketID)
}

func TestMapTransportSend(t *testing.T) {
	sess.GenerateID = func() tmap.SocketID { return tmap.SocketID("ABC123") }

	c := eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	wgrp := new(sync.WaitGroup)

	etr := eiot.NewPollingTransport(1000, 10*time.Millisecond)(tmap.SessionID("12345"), c)
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

	etr := eiot.NewPollingTransport(1000, 10*time.Millisecond)(tmap.SessionID("12345"), c)
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

// Test for an AckID
func TestTransportAckID(t *testing.T) {
	tsp := tmap.NewMapTransport(siop.NewPacketV2)

	// test to make sure the atomic incrementing works if we get
	// concurrent calls to AckID

	wg := new(sync.WaitGroup)
	ch := make(chan uint64, 100)
	wait := make(chan struct{})
	for i := 0; i < 100; i++ {

		wg.Add(1)
		go func() {
			<-wait // wait until we are done looping
			ch <- tsp.AckID()
			wg.Done()
		}()
	}
	close(wait) // now fire all go routines at the same time

	wg.Wait()
	close(ch)

	cm := map[uint64]struct{}{}
	for i := range ch {
		cm[i] = struct{}{}
	}

	if len(cm) != 100 {
		t.Log("got:", len(cm))
		t.Fatal("it looks like atomic didn't work as expected.")
	}
}

// Test for a MapTransport Set

type mockTransporter struct {
	eiot.Transporter
	id string
}

func (mt mockTransporter) ID() eiot.SessionID {
	return eiot.SessionID(mt.id)
}

func (mt mockTransporter) Receive() <-chan eiop.Packet {
	return nil
}

func newMockTransporter(id string) mockTransporter {
	return mockTransporter{id: id}
}

func TestTransportMapJoinLeaveRooms(t *testing.T) {
	type inline struct {
		name string
		idx  *int
		ids  []struct{ ns, rm, sio, eio string }
		gen  func(inline) func() tmap.SocketID
		mix  func(*testing.T, siot.Transporter, inline)
		eval func(*testing.T, siot.Transporter, inline)
	}
	var tests = []inline{
		{
			name: "basic join",
			ids: []struct {
				ns, rm, sio, eio string
			}{
				{"/", "101", "sio:abc123", "eio:abc123"},
				{"/", "101", "sio:def456", "eio:def456"},
				{"/", "101", "sio:ghi789", "eio:ghi789"},
				{"/", "101", "sio:jkl012", "eio:jkl012"},
				{"/", "101", "sio:mno345", "eio:mno345"},
			},
			mix: func(t *testing.T, tsp siot.Transporter, in inline) {
				for ; *in.idx < len(in.ids); *in.idx++ {
					sid, err := tsp.Add(newMockTransporter(in.ids[*in.idx].eio))
					assert.NoError(t, err)

					ns := in.ids[*in.idx].ns
					room := tmap.Room(in.ids[*in.idx].rm)
					err = tsp.Join(ns, sid, room)
					assert.NoError(t, err)
				}
			},
			eval: func(t *testing.T, tsp siot.Transporter, in inline) {
				ids, err := tsp.(siot.Emitter).Sockets("/").FromRoom("101")
				assert.NoError(t, err)

				var sessids []sess.ID
				for _, v := range in.ids {
					sessids = append(sessids, sess.ID(v.sio))
				}
				assert.ElementsMatch(t, sessids, ids)
			},
		},
		{
			name: "basic leave",
			ids: []struct {
				ns, rm, sio, eio string
			}{
				{"/", "101", "sio:abc123", "eio:abc123"},
				{"/", "101", "sio:def456", "eio:def456"},
				{"/", "101", "sio:ghi789", "eio:ghi789"},
				{"/", "101", "sio:jkl012", "eio:jkl012"},
				{"/", "101", "sio:mno345", "eio:mno345"},
			},
			mix: func(t *testing.T, tsp siot.Transporter, in inline) {
				var sids []sess.ID
				for ; *in.idx < len(in.ids); *in.idx++ {
					sid, err := tsp.Add(newMockTransporter(in.ids[*in.idx].eio))
					assert.NoError(t, err)

					ns := in.ids[*in.idx].ns
					room := tmap.Room(in.ids[*in.idx].rm)
					err = tsp.Join(ns, sid, room)
					assert.NoError(t, err)

					sids = append(sids, sid)
				}

				for i, sid := range sids {
					ns := in.ids[i].ns
					room := tmap.Room(in.ids[i].rm)
					err := tsp.Leave(ns, sid, room)
					assert.NoError(t, err)
				}
			},
			eval: func(t *testing.T, tsp siot.Transporter, in inline) {
				ids, err := tsp.(siot.Emitter).Sockets("/").FromRoom("101")
				assert.NoError(t, err)
				assert.Empty(t, ids)
			},
		},
		{
			name: "basic join with namespace",
			ids: []struct {
				ns, rm, sio, eio string
			}{
				{"/", "101", "sio:abc123", "eio:abc123"},
				{"/odd", "101", "sio:def456", "eio:def456"},
				{"/", "101", "sio:ghi789", "eio:ghi789"},
				{"/odd", "101", "sio:jkl012", "eio:jkl012"},
				{"/", "101", "sio:mno345", "eio:mno345"},
			},
			mix: func(t *testing.T, tsp siot.Transporter, in inline) {
				for ; *in.idx < len(in.ids); *in.idx++ {
					sid, err := tsp.Add(newMockTransporter(in.ids[*in.idx].eio))
					assert.NoError(t, err)

					ns := in.ids[*in.idx].ns
					room := tmap.Room(in.ids[*in.idx].rm)
					err = tsp.Join(ns, sid, room)
					assert.NoError(t, err)
				}
			},
			eval: func(t *testing.T, tsp siot.Transporter, in inline) {
				ids, err := tsp.(siot.Emitter).Sockets("/").FromRoom("101")
				assert.NoError(t, err)

				var sessids []sess.ID
				for i, v := range in.ids {
					if i%2 == 0 {
						sessids = append(sessids, sess.ID(v.sio))
					}
				}
				assert.ElementsMatch(t, sessids, ids)
			},
		},
		{
			name: "rooms",
			ids: []struct{ ns, rm, sio, eio string }{
				{"/", "101", "sio:abc123", "eio:abc123"},
				{"/", "102", "sio:abc123", "eio:abc123"},
				{"/", "103", "sio:abc123", "eio:abc123"},
				{"/", "104", "sio:abc123", "eio:abc123"}, // leave

				{"/", "101", "sio:def456", "eio:def456"},
				{"/", "102", "sio:def456", "eio:def456"},
				{"/skip", "103", "sio:def456", "eio:def456"},
				{"/", "104", "sio:def456", "eio:def456"},

				{"/", "101", "sio:ghi789", "eio:ghi789"},
				{"/aaa", "101", "sio:ghi789", "eio:ghi789"},
				{"/bbb", "101", "sio:ghi789", "eio:ghi789"},
				{"/bbb", "102", "sio:ghi789", "eio:ghi789"},
				{"/ccc", "101", "sio:ghi789", "eio:ghi789"},
				{"/ccc", "102", "sio:ghi789", "eio:ghi789"}, // leave
				{"/ccc", "103", "sio:ghi789", "eio:ghi789"},
			},
			mix: func(t *testing.T, tsp siot.Transporter, in inline) {
				for ; *in.idx < len(in.ids); *in.idx++ {
					sid, err := tsp.Add(newMockTransporter(in.ids[*in.idx].eio))
					assert.NoError(t, err)
					ns := in.ids[*in.idx].ns
					room := tmap.Room(in.ids[*in.idx].rm)
					err = tsp.Join(ns, sid, room)
					assert.NoError(t, err)
				}

				for _, i := range []int{3, 13} {
					ns := in.ids[i].ns
					sid := tmap.SocketID(in.ids[i].sio)
					room := in.ids[i].rm
					err := tsp.Leave(ns, sid, room)
					assert.NoError(t, err)
				}
			},
			eval: func(t *testing.T, tsp siot.Transporter, in inline) {
				var want = map[string][]struct {
					ns string
					rm []string
				}{
					"sio:abc123": {{"/", []string{"101", "102", "103"}}},
					"sio:def456": {{"/", []string{"101", "102", "104"}}},
					"sio:ghi789": {
						{"/", []string{"101"}},
						{"/bbb", []string{"101", "102"}},
						{"/ccc", []string{"101", "103"}},
					},
				}
				for k, vs := range want {
					sid := sess.ID(k)
					for _, v := range vs {
						rooms := tsp.(siot.Emitter).Rooms(v.ns, sid).Rooms
						assert.ElementsMatch(t, rooms, v.rm)
					}
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.idx == nil {
				test.idx = func() *int { var i int; return &i }()
			}
			if test.gen == nil {
				test.gen = func(in inline) func() tmap.SocketID {
					return func() tmap.SocketID {
						return tmap.SocketID(in.ids[*in.idx].sio)
					}
				}
			}

			sess.GenerateID = test.gen(test)
			tsp := tmap.NewMapTransport(siop.NewPacketV2)
			test.mix(t, tsp, test)
			test.eval(t, tsp, test)
		})
	}
}
