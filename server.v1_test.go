package socketio_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/njones/socketio"
	"github.com/njones/socketio/callback"
	"github.com/njones/socketio/serialize"
	"github.com/njones/socketio/session"
	"github.com/stretchr/testify/assert"
)

func TestServerStatus(t *testing.T) {
	svr := socketio.NewServerV4()

	req, err := http.NewRequest("GET", "/socket.io/?transport=polling", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	svr.ServeHTTP(rec, req)

	t.Error(rec.Result().StatusCode)
}

func TestServerV1(t *testing.T) {
	var opts = []func(*testing.T){runTest("sending to the client.Websocket")}
	var EIOv = 2

	runWithOptions := map[string]testParamsInFn{
		"Polling": func(v1 socketio.Server, count int, seq map[string][][]string, syncOn *sync.WaitGroup) testFn {
			return PollingTestV1(opts, EIOv, v1, count, seq, syncOn)
		},
		"Websocket": func(v1 socketio.Server, count int, seq map[string][][]string, syncOn *sync.WaitGroup) testFn {
			return WebsocketTestV1(opts, EIOv, v1, count, seq, syncOn)
		},
	}

	integration := map[string]testParamsOutFn{
		// spec - https://socket.io/docs/v2/emit-cheatsheet/
		"sending to the client":                                                   SendingToTheClientV1,
		"sending to all clients except sender":                                    SendingToAllClientsExceptTheSenderV1,
		"sending to all clients in 'game' room except sender":                     SendingToAllClientsInGameRoomExceptSenderV1,
		"sending to all clients in 'game1' and/or in 'game2' room, except sender": SendingToAllClientsInGame1AndOrGam2RoomExceptSenderV1,
		"sending to all clients in 'game' room, including sender":                 SendingToAllClientsInGameRoomIncludingSenderV1,
		"sending to all clients in namespace 'myNamespace', including sender":     SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV1,
		"sending to a specific room in a specific namespace, including sender":    SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV1,
		"sending to individual socketid (private message)":                        SendingToIndividualSocketIDPrivateMessageV1,
		"sending with acknowledgement":                                            SendingWithAcknowledgementV1,
		"sending to all connected clients":                                        SendingToAllConnectedClientsV1,

		// extra tests outside of the spec based ones...
		"on event":          OnEventV1,
		"reject the client": RejectTheClientV1,
	}

	for name, testParams := range integration {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}
}

func PollingTestV1(opts []func(*testing.T), EIOv int, vX socketio.Server, count int, seq map[string][][]string, syncOn *sync.WaitGroup) func(*testing.T) {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		t.Parallel()

		var (
			server = httptest.NewServer(vX)
			client = make([]*testClient, count)
		)

		defer server.Close()

		// send onConnect events (auto onConnect for v2)
		for i := 0; i < count; i++ {
			client[i] = &testClient{polling: &v1PollingClient{
				t:          t,
				base:       server.URL,
				client:     server.Client(),
				buffer:     new(bytes.Buffer),
				eioVersion: EIOv,
			}}

			var queryStr []string
			if q, ok := seq["connect_query"]; ok {
				queryStr = q[i]
			}
			client[i].polling.connect(queryStr)
		}

		// wait for all onConnection events to complete...
		syncOn.Wait()

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

					for _, packet := range packets {
						packetBuf.WriteString(fmt.Sprintf("%d:%s", len(packet), packet))
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

		// wait for all emitted events to complete...
		syncOn.Wait()

		// check that we hit every "send/grab" that we needed to check...
		if xq, ok := seq["connect_query"]; ok {
			x += len(xq)
		}
		assert.Equal(t, count*len(seq), x)
	}
}

func WebsocketTestV1(opts []func(*testing.T), EIOv int, vX socketio.Server, count int, seq map[string][][]string, syncOn *sync.WaitGroup) func(*testing.T) {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		t.Parallel()

		var (
			server = httptest.NewServer(vX)
			client = make([]*testClient, count)
		)

		defer server.Close()

		// send onConnect events (auto onConnect for v2)
		for i := 0; i < count; i++ {
			client[i] = &testClient{websocket: &v1WebsocketClient{
				t:          t,
				base:       server.URL,
				client:     server.Client(),
				buffer:     new(bytes.Buffer),
				eioVersion: EIOv,
			}}

			var queryStr []string
			if q, ok := seq["connect_query"]; ok {
				queryStr = q[i]
			}
			client[i].websocket.connect(queryStr)
		}

		// wait for all onConnection events to complete...
		syncOn.Wait()

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

					for _, packet := range packets {
						packetBuf.WriteString(fmt.Sprintf("%d:%s", len(packet), packet))
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

		// wait for all emitted events to complete...
		syncOn.Wait()

		// check that we hit every "send/grab" that we needed to check...
		if xq, ok := seq["connect_query"]; ok {
			x += len(xq)
		}
		assert.Equal(t, count*len(seq), x)
	}
}

func SendingToTheClientV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
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

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		socket.Emit("hello", str("can you hear me?"), one, serialize.Integer(2), str("abc"))
		return nil
	})

	return v1, count, want, wait
}

func SendingToAllClientsExceptTheSenderV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["broadcast","Hello friends!"]`},
				{`42["broadcast","Hello friends!"]`},
				nil,
			},
		}
		count = len(want["grab1"])
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		if cnt == (count - 1) {
			socket.Broadcast().Emit("broadcast", serialize.String("Hello friends!"))
		}

		cnt++
		return nil
	})

	return v1, count, want, wait
}

func SendingToAllClientsInGameRoomExceptSenderV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
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
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		if cnt%2 == 0 {
			socket.Join("game")
		}

		if cnt == (count - 1) {
			socket.To("game").Emit("nice game", serialize.String("let's play a game"))
		}
		cnt++
		return nil
	})

	return v1, count, want, wait
}

func SendingToAllClientsInGame1AndOrGam2RoomExceptSenderV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
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
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		if cnt%2 == 0 {
			socket.Join("game1")
		}
		if cnt%3 == 0 {
			socket.Join("game2")
		}

		if cnt == (count - 1) {
			socket.In("game1").To("game2").Emit("nice game", serialize.String("let's play a game (too)"))
		}
		cnt++
		return nil
	})

	return v1, count, want, wait
}

func SendingToAllClientsInGameRoomIncludingSenderV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
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
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		if cnt%2 == 0 {
			socket.Join("game")
		}

		if cnt == (count - 1) {
			v1.To("game").Emit("big-announcement", serialize.String("the game will start soon"))
		}
		cnt++
		return nil
	})

	return v1, count, want, wait
}

func SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"send1": {
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
				{`42/myNamespace,["bigger-announcement","the tournament will start soon"]`}, // 42/myNamespace,["bigger-announcement","the tournament will start soon"]
			},
		}
		count = len(want["send1"])
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		return nil
	})

	var nsCount int
	for _, v := range want["send1"] {
		if v != nil {
			nsCount++
		}
	}

	v1.Of("myNamespace").OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()
		cnt++

		if cnt == nsCount {
			v1.Of("myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
		}
		return nil
	})

	return v1, count, want, wait
}

func SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"send1": {
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
		count = len(want["send1"])
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		socket.Join("room")
		return nil
	})

	var nsCount int
	for _, v := range want["send1"] {
		if v != nil {
			nsCount++
		}
	}

	v1.Of("myNamespace").OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()
		cnt++

		if cnt == 1 {
			socket.Join("room")
		}

		if cnt == nsCount {
			socket.Join("room")
			v1.Of("myNamespace").To("room").Emit("event", serialize.String("message"))
		}

		return nil
	})

	return v1, count, want, wait
}

func SendingToIndividualSocketIDPrivateMessageV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {
				{`42["hey","I just met you #1"]`},
				{`42["hey","I just met you #2"]`},
				{`42["hey","I just met you #0"]`},
			},
		}
		count = len(want["grab1"])
		cnt   = 0
	)

	nextSocketID := make([]chan session.ID, count)
	for i := range nextSocketID {
		nextSocketID[i] = make(chan session.ID, 1)
	}

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		nextSocketID[(cnt+1)%count] <- socket.ID()

		go func(num int) {
			defer wait.Done()

			socketID := <-nextSocketID[num]
			v1.In(string(socketID)).Emit("hey", serialize.String(fmt.Sprintf("I just met you #%d", num)))
		}(cnt)

		cnt++
		return nil
	})

	return v1, count, want, wait
}

func SendingWithAcknowledgementV1(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"grab1": {{`421["question","do you think so?"]`}},
			"send2": {{`431["answer",42]`}},
		}
		count = len(want["grab1"])
		cnt   = 0
	)

	var question = serialize.String("do you think so?")

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
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

		cnt++
		return nil
	})

	return v1, count, want, wait
}

func SendingToAllConnectedClientsV1(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
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
		cnt   = 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		if cnt == (count - 1) {
			v1.Emit("*", serialize.String("an event sent to all connected clients"))
		}
		cnt++
		return nil
	})

	return v1, count, want, wait
}

func OnEventV1(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
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
		count  = len(want["send1"])
		cnt, n = 0, 0
	)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()

		socket.Join("room")

		socket.On("chat message", callback.Wrap{
			Parameters: []serialize.Serializable{serialize.StrParam},
			Func: func() interface{} {

				return func(msg string) error {
					defer wait.Done()

					if n%2 != 0 {
						socket.Leave("room")
					}

					assert.Equal(t, fmt.Sprintf("an event sent to all connected clients #%d", n), msg)
					n++
					return nil
				}
			},
		})

		v1.OnDisconnect(func(reason string) {
			defer wait.Done()

			v1.In("room").Emit("say goodbye", serialize.String("disconnecting..."))
		})

		cnt++
		return nil
	})

	return v1, count, want, wait
}

func RejectTheClientV1(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v1   = socketio.NewServerV1(testingQuickPoll)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"connect_query": {{`access=true`}, {`access=false`}},
			"grab1":         {{`42["hello",1]`}, {`44{"message":"not authorized"}`}},
		}
		count = len(want["connect_query"])
		cnt   = 0
	)

	checkCount(t, count)

	wait.Add(count)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer wait.Done()
		cnt++

		tf := socket.Request().URL.Query().Get("access")
		if tf == "true" {
			socket.Emit("hello", serialize.Integer(1))
			return nil
		}

		return fmt.Errorf("not authorized")
	})

	return v1, count, want, wait
}
