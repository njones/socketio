package socketio_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/njones/socketio"
	"github.com/njones/socketio/callback"
	"github.com/njones/socketio/engineio"
	"github.com/njones/socketio/serialize"
	"github.com/stretchr/testify/assert"
)

var testingOptionsV4 = []socketio.Option{engineio.WithPingTimeout(1 * time.Second), engineio.WithPingInterval(500 * time.Millisecond)}

func TestServerV4(t *testing.T) {
	var opts = []func(*testing.T){}
	var EIOv = 4

	runWithOptions := map[string]testParamsInFn{
		"Polling": func(v4 socketio.Server, count int, want map[string][][]string, wait *sync.WaitGroup) testFn {
			return PollingTestV4(opts, EIOv, v4, count, want, wait)
		},
		"Websocket": func(v4 socketio.Server, count int, want map[string][][]string, wait *sync.WaitGroup) testFn {
			return WebsocketTestV3(opts, EIOv, v4, count, want, wait)
		},
	}

	integration := map[string]testParamsOutFn{
		// spec - https://socket.io/docs/v3/emit-cheatsheet/
		"sending to the client":                                                   SendingToTheClientV4,
		"sending to all clients except sender":                                    SendingToAllClientsExceptTheSenderV4,
		"sending to all clients in 'game' room except sender":                     SendingToAllClientsInGameRoomExceptSenderV4,
		"sending to all clients in 'game1' and/or in 'game2' room, except sender": SendingToAllClientsInGame1AndOrGame2RoomExceptSenderV4,
		"sending to all clients in 'game' room, including sender":                 SendingToAllClientsInGameRoomIncludingSenderV4,
		"to all clients in room1 and/or room2 except those in room3":              SendingToAllClientsInRoom1AndOrRoom2ExceptRoom3V4,
		"sending to all clients in namespace 'myNamespace', including sender":     SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV4,
		"sending to a specific room in a specific namespace, including sender":    SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV4,
		"sending to individual socketid (private message)":                        SendingToIndividualSocketIDPrivateMessageV4,
		"sending with acknowledgement":                                            SendingWithAcknowledgementV4,
		"sending to all connected clients":                                        SendingToAllConnectedClientsV4,

		// extra
		"on event":                                   OnEventV4,
		"reject the client":                          RejectTheClientV4,
		"sending a binary event from the client":     SendingBinaryEventFromClientV4,
		"sending a binary ack event from the client": SendingBinaryAckFromClientV4,
	}

	for name, testParams := range integration {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func PollingTestV4(opts []func(*testing.T), EIOv int, v4 socketio.Server, count int, seq map[string][][]string, syncOn *sync.WaitGroup) testFn {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		t.Parallel()

		var (
			server = httptest.NewServer(v4)
			client = make([]*testClient, count)
		)

		defer server.Close()

		for i := 0; i < count; i++ {
			client[i] = &testClient{polling: &v3PollingClient{
				t:          t,
				base:       server.URL,
				client:     server.Client(),
				buffer:     new(bytes.Buffer),
				eioVersion: EIOv,
			}}

			var queryStr, connStr []string

			if q, ok := seq["connect_query"]; ok {
				queryStr = q[i]
			}
			if c, ok := seq["connect"]; ok {
				connStr = c[i]
			}

			client[i].polling.connect(queryStr, connStr)
		}

		syncOn.Wait() // waits for all of the events inside of a onConnection to complete...

		var x int
		reqSequence := []string{"send1", "grab1", "send2", "grab2"}
		for _, reqType := range reqSequence {
			if request, ok := seq[reqType]; ok && strings.HasPrefix(reqType, "send") {
				var packetBuf = new(bytes.Buffer)

				for i, packets := range request {
					x++
					if len(packets) == 0 {
						continue
					}

					syncOn.Add(1)
					packetBuf.Reset()

					for i, packet := range packets {
						if i > 0 {
							packetBuf.WriteByte(0x1e)
						}
						packetBuf.WriteString(packet)
					}
					client[i].polling.send(packetBuf)
				}
				continue
			}
			if request, ok := seq[reqType]; ok && strings.HasPrefix(reqType, "grab") {
				for i, want := range request {
					x++
					have := client[i].polling.grab()
					assert.Equal(t, want, have, "[%s] idx: %d", reqType, i)
				}
				continue
			}
		}

		syncOn.Wait()

		if xq, ok := seq["connect_query"]; ok {
			x += len(xq)
		}
		if xc, ok := seq["connect"]; ok {
			x += len(xc)
		}

		assert.Equal(t, count*len(seq), x) // the wants were actually tested
	}
}

func SendingToTheClientV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["hello","can you hear me?",1,2,"abc"]`},
				{`42["hello","can you hear me?",1,2,"abc"]`},
				{`42["hello","can you hear me?",1,2,"abc"]`},
			},
		}
		count = len(want["grab1"])

		str = serialize.String
		one = serialize.Integer(1)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		socket.Emit("hello", str("can you hear me?"), one, serialize.Integer(2), str("abc"))
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllClientsExceptTheSenderV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["broadcast","Hello friends!"]`},
				{`42["broadcast","Hello friends!"]`},
				nil,
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.Broadcast().Emit("broadcast", serialize.String("Hello friends!"))
		}

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllClientsInGameRoomExceptSenderV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["nice game","let's play a game"]`},
				nil,
				{`42["nice game","let's play a game"]`},
				nil,
				nil, // sender...
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt)%2 == 0 {
			socket.Join("game")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.To("game").Emit("nice game", serialize.String("let's play a game"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllClientsInGame1AndOrGame2RoomExceptSenderV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["nice game","let's play a game (too)"]`},
				nil,
				{`42["nice game","let's play a game (too)"]`},
				{`42["nice game","let's play a game (too)"]`},
				{`42["nice game","let's play a game (too)"]`},
				nil,
				{`42["nice game","let's play a game (too)"]`},
				nil,
				{`42["nice game","let's play a game (too)"]`},
				{`42["nice game","let's play a game (too)"]`},
				{`42["nice game","let's play a game (too)"]`},
				nil,
				nil, // sender...
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt)%2 == 0 {
			socket.Join("game1")
		}
		if atomic.LoadInt64(&cnt)%3 == 0 {
			socket.Join("game2")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.In("game1", "game2").Emit("nice game", serialize.String("let's play a game (too)"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllClientsInGameRoomIncludingSenderV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["big-announcement","the game will start soon"]`},
				nil,
				{`42["big-announcement","the game will start soon"]`},
				nil,
				{`42["big-announcement","the game will start soon"]`},
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt)%2 == 0 {
			socket.Join("game")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v4.To("game").Emit("big-announcement", serialize.String("the game will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllClientsInRoom1AndOrRoom2ExceptRoom3V4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				nil, // 0 0 [0] // 00 -- (%2 %3 %4)
				nil, // 1 1 1   // 01
				{`42["nice game","let's play a game (too)"]`}, // 0 2 2 // 02
				{`42["nice game","let's play a game (too)"]`}, // 1 0 3 // 03
				nil, // 0 1 [0] // 04
				nil, // 1 2 1   // 05
				{`42["nice game","let's play a game (too)"]`}, // 0 0 2 // 06
				nil, // 1 1 3   // 07
				nil, // 0 2 [0] // 08
				{`42["nice game","let's play a game (too)"]`}, // 1 0 1 // 09
				{`42["nice game","let's play a game (too)"]`}, // 0 1 2 // 10
				nil, // 1 2 3   // 11
				nil, // 0 0 [0] // 12
				nil, // 1 1 1   // 13
				{`42["nice game","let's play a game (too)"]`}, // 0 2 2 // 14
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt)%2 == 0 {
			socket.Join("room1")
		}
		if atomic.LoadInt64(&cnt)%3 == 0 {
			socket.Join("room2")
		}
		if atomic.LoadInt64(&cnt)%4 == 0 {
			socket.Join("room3")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v4.Except("room3").To("room1", "room2").Emit("nice game", serialize.String("let's play a game (too)"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"connect": {
				{`40/myNamespace,`},
				nil,
				{`40/myNamespace,`},
				nil,
				{`40/myNamespace,`},
			},
			"grab1": {
				{`42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
				nil,
				{`42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
				nil,
				{`42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
			},
		}
		count = len(want["connect"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v4.Of("myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v4.Of("myNamespace").OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v4.Of("myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"connect": {
				{`40/myNamespace,`},
				nil,
				{`40/myNamespace,`},
				nil,
				{`40/myNamespace,`},
			},
			"grab1": {
				{`42/myNamespace,["event","message"]`},
				nil,
				nil,
				nil,
				{`42/myNamespace,["event","message"]`},
			},
		}
		count = len(want["connect"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		socket.Join("room")
		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v4.Of("myNamespace").To("room").Emit("event", serialize.String("message"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v4.Of("myNamespace").OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == 0 {
			socket.Join("room")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.Join("room")
			v4.Of("myNamespace").To("room").Emit("event", serialize.String("message"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToIndividualSocketIDPrivateMessageV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["hey","I just met you #1"]`},
				{`42["hey","I just met you #2"]`},
				{`42["hey","I just met you #0"]`},
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	nextSocketID := make([]chan string, count)
	for i := range nextSocketID {
		nextSocketID[i] = make(chan string, 1)
	}

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		nextSocketID[(int(atomic.LoadInt64(&cnt)+1) % count)] <- socket.ID().String()

		go func(num int) {
			defer wait.Done()

			socketID := <-nextSocketID[num]
			v4.In(socketID).Emit("hey", serialize.String(fmt.Sprintf("I just met you #%d", num)))
		}(int(atomic.LoadInt64(&cnt)))

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingWithAcknowledgementV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {{`421["question","do you think so?"]`}},
			"send2": {{`431["answer",42]`}},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	var question = serialize.String("do you think so?")

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		wait.Done()

		err := socket.Emit("question", question, callback.Wrap{
			Parameters: []serialize.Serializable{serialize.StrParam, serialize.IntParam},
			Func: func() interface{} {
				return func(value1 string, value2 int) error {
					wait.Done()

					assert.Equal(t, "answer", value1)
					assert.Equal(t, 42, value2)

					return nil
				}
			},
		})

		assert.NoError(t, err)

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func SendingToAllConnectedClientsV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["*","an event sent to all connected clients"]`},
				{`42["*","an event sent to all connected clients"]`},
				{`42["*","an event sent to all connected clients"]`},
				{`42["*","an event sent to all connected clients"]`},
				{`42["*","an event sent to all connected clients"]`},
			},
		}
		count = len(want["grab1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v4.Emit("*", serialize.String("an event sent to all connected clients"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}

func OnEventV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"send1": {
				{`42["chat message","an event sent to all connected clients #0"]`},
				{`42["chat message","an event sent to all connected clients #1"]`},
				{`42["chat message","an event sent to all connected clients #2"]`},
				{`42["chat message","an event sent to all connected clients #3"]`},
				{`42["chat message","an event sent to all connected clients #4"]`},
			},
			"send2": {
				{`41`}, nil, nil, nil, nil,
			},
			"grab2": {
				{`42["say goodbye","disconnecting..."]`},
				nil,
				{`42["say goodbye","disconnecting..."]`},
				nil,
				{`42["say goodbye","disconnecting..."]`},
			},
		}
		count = len(want["send1"])
		n     = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		socket.Join("room")
		socket.On("chat message", callback.Wrap{
			Parameters: []serialize.Serializable{serialize.StrParam},
			Func: func() interface{} {

				return func(msg string) error {
					defer wait.Done()

					if atomic.LoadInt64(&n)%2 != 0 {
						socket.Leave("room")
					}

					assert.Equal(t, fmt.Sprintf("an event sent to all connected clients #%d", n), msg)
					atomic.AddInt64(&n, 1)
					return nil
				}
			},
		})

		v4.OnDisconnect(func(reason string) {
			defer wait.Done()

			v4.In("room").Emit("say goodbye", serialize.String("disconnecting..."))
		})

		return nil
	})

	return v4, count, want, wait
}

func RejectTheClientV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"connect_query": {{`access=true`}, {`access=false`}},
			"grab1":         {{`42["hello",1]`}, {`44{"message":"not authorized"}`}},
		}
		count = len(want["connect_query"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		atomic.AddInt64(&cnt, 1)

		tf := socket.Request().URL.Query().Get("access")
		if tf == "true" {
			socket.Emit("hello", serialize.Integer(1))
			return nil
		}

		return fmt.Errorf("not authorized")
	})

	return v4, count, want, wait
}

func SendingBinaryEventFromClientV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"send1": {
				{`451-["hello",{"_placeholder":true,"num":0}]`, `bAQIDBA==`},
			},
		}
		count  = len(want["send1"])
		cnt    = int64(0)
		expect = []byte{0x01, 0x02, 0x03, 0x04}
	)

	checkCount(t, count)

	wait.Add(count)
	v4.OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v4.On("hello", testBinaryEventFunc(func(r io.Reader) {
		defer wait.Done()

		have, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, expect, have)
	}))

	return v4, count, want, wait
}

func SendingBinaryAckFromClientV4(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v4   = socketio.NewServerV4(testingOptionsV4...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"connect": {{`40/admin,`}},
			"grab1": {
				{`42/admin,1["question","binary data?"]`},
			},
			"send2": {
				{`461-/admin,1[{"_placeholder":true,"num":0}]`, `bAQIDBA==`},
			},
		}
		count  = len(want["connect"])
		cnt    = int64(0)
		expect = []byte{0x01, 0x02, 0x03, 0x04}
	)

	checkCount(t, count)

	var question = serialize.String("binary data?")

	wait.Add(count)
	v4.Of("/admin").OnConnect(func(socket *socketio.SocketV4) error {
		defer wait.Done()

		err := socket.Emit("question", question, callback.Wrap{
			Parameters: []serialize.Serializable{serialize.BinParam},
			Func: func() interface{} {
				return func(value1 io.Reader) error {
					wait.Done()

					have, err := io.ReadAll(value1)
					assert.NoError(t, err)
					assert.Equal(t, expect, have)

					return nil
				}
			},
		})

		assert.NoError(t, err)

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v4, count, want, wait
}
