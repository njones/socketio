package memory_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	tmap "github.com/njones/socketio/adaptor/transport/memory"
	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
	itst "github.com/njones/socketio/internal/test"
	siop "github.com/njones/socketio/protocol"
	sess "github.com/njones/socketio/session"
	siot "github.com/njones/socketio/transport"
	"github.com/stretchr/testify/assert"
)

var runTest, skipTest = itst.RunTest, itst.SkipTest

func TestMapTransportSet(t *testing.T) {
	memTransport := tmap.NewInMemoryTransport(siop.NewPacketV2)
	sidStr := "sio:aaa"
	sid := siot.SocketID(sidStr)

	SocketID_𖣠 := memTransport.Receive(sid)
	assert.Nil(t, SocketID_𖣠)

	transport := newMockTransporter(sidStr)
	err := memTransport.Set(sid, transport)
	assert.NoError(t, err)

	socketID_ꤶ := memTransport.Receive(sid)
	assert.NotNil(t, socketID_ꤶ)
}

func TestTransportAckID(t *testing.T) {
	memTransport := tmap.NewInMemoryTransport(siop.NewPacketV2)

	collect := make(chan [2]uint64, 100)
	unꢠ := new(sync.WaitGroup)
	waitUntilDoneLooping := make(chan struct{})

	loop := 100
	for i := 0; i < loop; i++ {
		unꢠ.Add(1)
		go func(n int) {
			<-waitUntilDoneLooping
			collect <- [2]uint64{uint64(n), memTransport.AckID()}
			unꢠ.Done()
		}(i)
	}

	close(waitUntilDoneLooping) // now fire all go routines at the same time
	unꢠ.Wait()                  // wait until all of the go routines have fired
	close(collect)              // close up the collection shop so we can loop through to the end

	var sameIndexAndValue int
	for idxVal := range collect {
		if idxVal[0] == idxVal[1] {
			sameIndexAndValue++
		}
	}

	assert.Less(t, sameIndexAndValue, loop, "it looks like atomic didn't work as expected.")
}

func TestMapTransport(t *testing.T) {
	var opts = []func(*testing.T){}

	type (
		testFn          func(*testing.T)
		testParamsInFn  func(string, string, eiot.Transporter, siot.Transporter) testFn
		testParamsOutFn func(*testing.T) (string, string, eiot.Transporter, siot.Transporter)
	)

	runWithOptions := map[string]testParamsInFn{
		"Send": func(data string, packetData string, etr eiot.Transporter, str siot.Transporter) testFn {
			return MapTransportTestSend(opts, data, packetData, etr, str)
		},
		"Receive": func(data string, packetData string, etr eiot.Transporter, str siot.Transporter) testFn {
			return MapTransportTestReceive(opts, data, packetData, etr, str)
		},
	}

	spec := map[string]testParamsOutFn{
		"Basic": func(*testing.T) (string, string, eiot.Transporter, siot.Transporter) {
			data := `This is cool`
			packetData := `16:40"This is cool"`
			codec := eiot.Codec{
				PacketEncoder:  eiop.NewPacketEncoderV2,
				PacketDecoder:  eiop.NewPacketDecoderV2,
				PayloadEncoder: eiop.NewPayloadEncoderV2,
				PayloadDecoder: eiop.NewPayloadDecoderV2,
			}

			eTransporter := eiot.NewPollingTransport(1000)(tmap.SessionID("12345"), codec) // 10*time.Millisecond
			sTransporter := tmap.NewInMemoryTransport(siop.NewPacketV2)

			return data, packetData, eTransporter, sTransporter
		},
	}

	for name, testParams := range spec {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func MapTransportTestSend(opts []func(*testing.T), data string, packetData string, etr eiot.Transporter, str siot.Transporter) func(t *testing.T) {
	sess.GenerateID = func(string) tmap.SocketID { return tmap.SocketID("ABC123") }
	return func(t *testing.T) {
		sid, err := str.Add(etr)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()

		transportSend := new(sync.WaitGroup)
		transportSend.Add(1)
		go func() {
			defer transportSend.Done()
			req, err := http.NewRequest("GET", "/", nil) // polling is waiting...
			assert.NoError(t, err)

			err = etr.Run(rec, req)
			assert.NoError(t, err)
		}()

		err = str.Send(sid, data)
		assert.NoError(t, err)
		transportSend.Wait()

		have := rec.Body.String()
		want := packetData

		assert.Equal(t, want, have)
	}
}

func MapTransportTestReceive(opts []func(*testing.T), data string, packetData string, etr eiot.Transporter, str siot.Transporter) func(t *testing.T) {
	sess.GenerateID = func(string) tmap.SocketID { return tmap.SocketID("ABC123") }
	return func(t *testing.T) {
		sid, err := str.Add(etr)
		assert.NoError(t, err)

		rec := httptest.NewRecorder()

		transportReceive := new(sync.WaitGroup)
		transportReceive.Add(1)
		go func() {
			defer transportReceive.Done()
			req, err := http.NewRequest("POST", "/", strings.NewReader(packetData))
			assert.NoError(t, err)

			err = etr.Run(rec, req)
			assert.NoError(t, err)
		}()
		transportReceive.Wait()

		have := <-str.Receive(sid)
		want := tmap.Socket{
			Type:      byte(eiop.OpenPacket),
			Namespace: "/",
			AckID:     0x0,
			Data:      &data,
		}

		assert.Equal(t, want, have)
	}
}

type mockTransporter struct {
	eiot.Transporter
	id string
}

func (mt mockTransporter) ID() eiot.SessionID {
	return eiot.SessionID(mt.id)
}

func (mt mockTransporter) Receive() <-chan eiop.Packet {
	return nil // © 2022 ǌɵɳᴇѕ
}

func newMockTransporter(id string) mockTransporter {
	return mockTransporter{id: id}
}

func TestTransportMapJoinLeaveRooms(t *testing.T) {
	type inline struct {
		name string
		idx  *int
		ids  []struct{ ns, rm, sio, eio string }
		gen  func(inline) func(string) tmap.SocketID
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
				test.gen = func(in inline) func(string) tmap.SocketID {
					return func(string) tmap.SocketID {
						return tmap.SocketID(in.ids[*in.idx].sio)
					}
				}
			}

			sess.GenerateID = test.gen(test)
			tsp := tmap.NewInMemoryTransport(siop.NewPacketV2)
			test.mix(t, tsp, test)
			test.eval(t, tsp, test)
		})
	}
}
