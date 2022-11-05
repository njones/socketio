package socketio_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
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
	"github.com/njones/socketio/session"
	"github.com/stretchr/testify/assert"
)

var testingOptionsV2 = []socketio.Option{
	engineio.WithSessionShave(1 * time.Millisecond),
	engineio.WithPingTimeout(2 * time.Second),
	engineio.WithPingInterval(500 * time.Millisecond),
}

func TestServerV2(t *testing.T) {
	var opts = []func(*testing.T){}
	var EIOv = 3

	runWithOptions := map[string]testParamsInFn{
		"Polling": func(v2 socketio.Server, count int, want map[string][][]string, wait *sync.WaitGroup) testFn {
			return PollingTestV2(opts, EIOv, v2, count, want, wait)
		},
		"Websocket": func(v2 socketio.Server, count int, want map[string][][]string, wait *sync.WaitGroup) testFn {
			return WebsocketTestV1(opts, EIOv, v2, count, want, wait)
		},
	}

	integration := map[string]testParamsOutFn{
		"sending to the client":                                                   SendingToTheClientV2,
		"sending to all clients except sender":                                    SendingToAllClientsExceptTheSenderV2,
		"sending to all clients in 'game' room except sender":                     SendingToAllClientsInGameRoomExceptSenderV2,
		"sending to all clients in 'game1' and/or in 'game2' room, except sender": SendingToAllClientsInGame1AndOrGame2RoomExceptSenderV2,
		"sending to all clients in 'game' room, including sender":                 SendingToAllClientsInGameRoomIncludingSenderV2,
		"sending to all clients in namespace 'myNamespace', including sender":     SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV2,
		"sending to a specific room in a specific namespace, including sender":    SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV2,
		"sending to individual socketid (private message)":                        SendingToIndividualSocketIDPrivateMessageV2,
		"sending with acknowledgement":                                            SendingWithAcknowledgementV2,
		"sending to all connected clients":                                        SendingToAllConnectedClientsV2,

		// extra
		"on event":                               OnEventV2,
		"reject the client":                      RejectTheClientV2,
		"sending a binary event from the client": SendingBinaryEventFromClientV2,
	}

	for name, testParams := range integration {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}

}

func PollingTestV2(opts []func(*testing.T), EIOv int, v2 socketio.Server, count int, seq map[string][][]string, syncOn *sync.WaitGroup) testFn {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		t.Parallel()

		var (
			server = httptest.NewServer(v2)
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

		// check that we hit every "want" that we needed to check...
		if xq, ok := seq["connect_query"]; ok {
			x += len(xq)
		}
		assert.Equal(t, count*len(seq), x)
	}
}

func SendingToTheClientV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		socket.Emit("hello", str("can you hear me?"), one, serialize.Integer(2), str("abc"))
		return nil
	})

	return v2, count, want, wait
}

func SendingToAllClientsExceptTheSenderV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.Broadcast().Emit("broadcast", serialize.String("Hello friends!"))
		}

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v2, count, want, wait
}

func SendingToAllClientsInGameRoomExceptSenderV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		if cnt%2 == 0 {
			socket.Join("game")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.To("game").Emit("nice game", serialize.String("let's play a game"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v2, count, want, wait
}

func SendingToAllClientsInGame1AndOrGame2RoomExceptSenderV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		if cnt%2 == 0 {
			socket.Join("game1")
		}
		if cnt%3 == 0 {
			socket.Join("game2")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.In("game1").To("game2").Emit("nice game", serialize.String("let's play a game (too)"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v2, count, want, wait
}

func SendingToAllClientsInGameRoomIncludingSenderV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		if cnt%2 == 0 {
			socket.Join("game")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v2.To("game").Emit("big-announcement", serialize.String("the game will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v2, count, want, wait
}

func SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
				{`42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
			},
		}
		count = len(want["send1"])
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		return nil
	})

	var nsCount int64
	for _, v := range want["send1"] {
		if v != nil {
			atomic.AddInt64(&nsCount, 1)
		}
	}

	v2.Of("myNamespace").OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()
		atomic.AddInt64(&cnt, 1)

		if atomic.LoadInt64(&cnt) == atomic.LoadInt64(&nsCount) {
			v2.Of("myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
		}
		return nil
	})

	return v2, count, want, wait
}

func SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
		cnt   = int64(0)
	)

	checkCount(t, count)

	wait.Add(count)
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		socket.Join("room")
		return nil
	})

	var nsCount int64
	for _, v := range want["send1"] {
		if v != nil {
			atomic.AddInt64(&nsCount, 1)
		}
	}

	v2.Of("myNamespace").OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()
		atomic.AddInt64(&cnt, 1)

		if atomic.LoadInt64(&cnt) == 1 {
			socket.Join("room")
		}

		if atomic.LoadInt64(&cnt) == atomic.LoadInt64(&nsCount) {
			socket.Join("room")
			v2.Of("myNamespace").To("room").Emit("event", serialize.String("message"))
		}

		return nil
	})

	return v2, count, want, wait
}

func SendingToIndividualSocketIDPrivateMessageV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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

	nextSocketID := make([]chan session.ID, count)
	for i := range nextSocketID {
		nextSocketID[i] = make(chan session.ID, 1)
	}

	wait.Add(count)
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		nextSocketID[int(atomic.LoadInt64(&cnt)+1)%count] <- socket.ID()

		go func(num int) {
			defer wait.Done()

			socketID := <-nextSocketID[num]
			v2.In(string(socketID)).Emit("hey", serialize.String(fmt.Sprintf("I just met you #%d", num)))
		}(int(atomic.LoadInt64(&cnt)))

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v2, count, want, wait
}

func SendingWithAcknowledgementV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
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

	return v2, count, want, wait
}

func SendingToAllConnectedClientsV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v2.Emit("*", serialize.String("an event sent to all connected clients"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return v2, count, want, wait
}

func OnEventV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	log.Println("start....")
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	log.Println("stop....")

	checkCount(t, count)

	wait.Add(count)
	v2.OnConnect(func(socket *socketio.SocketV2) error {
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

		v2.OnDisconnect(func(reason string) {
			defer wait.Done()

			v2.In("room").Emit("say goodbye", serialize.String("disconnecting..."))
		})

		return nil
	})

	return v2, count, want, wait
}

func RejectTheClientV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
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
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()
		atomic.AddInt64(&cnt, 1)

		tf := socket.Request().URL.Query().Get("access")
		if tf == "true" {
			socket.Emit("hello", serialize.Integer(1))
			return nil
		}

		return fmt.Errorf("not authorized")
	})

	return v2, count, want, wait
}

func SendingBinaryEventFromClientV2(t *testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup) {
	var (
		v2   = socketio.NewServerV2(testingOptionsV2...)
		wait = new(sync.WaitGroup)

		want = map[string][][]string{
			"send1": {
				{`451-["hello",{"base64":true,"data":"xAQBAgME"}]`},
			},
		}
		count  = len(want["send1"])
		cnt    = int64(0)
		expect = []byte{0x01, 0x02, 0x03, 0x04}
	)

	checkCount(t, count)

	wait.Add(count)
	v2.OnConnect(func(socket *socketio.SocketV2) error {
		defer wait.Done()

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v2.On("hello", testBinaryEventFunc(func(r io.Reader) {
		defer wait.Done()

		have, err := io.ReadAll(r)
		assert.NoError(t, err)
		if !assert.Equal(t, expect, have) {
			t.FailNow()
		}
	}))

	return v2, count, want, wait
}
