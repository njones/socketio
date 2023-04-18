package socketio_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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

var testingOptionsV3 = []socketio.Option{
	engineio.WithSessionShave(1 * time.Millisecond),
	engineio.WithPingTimeout(1 * time.Second),
	engineio.WithPingInterval(500 * time.Millisecond),
}

func TestSocketIOPathV3(t *testing.T) {

	tests := map[string]struct {
		withPath string
		reqPath  string
		expect   string
	}{
		"no path": {
			withPath: "",
			reqPath:  "socket.io",
			expect:   `^(\d+\{.[^\}]*\})+$`,
		},
		"socket.io": {
			withPath: "socket.io",
			reqPath:  "socket.io",
			expect:   `^(\d+\{.[^\}]*\})+$`,
		},
		"socket.io with left slash": {
			withPath: "/socket.io",
			reqPath:  "socket.io",
			expect:   `^(\d+\{.[^\}]*\})+$`,
		},
		"socket.io with right slash": {
			withPath: "socket.io/",
			reqPath:  "socket.io",
			expect:   `^(\d+\{.[^\}]*\})+$`,
		},
		"socket.io with both slash": {
			withPath: "/socket.io/",
			reqPath:  "socket.io",
			expect:   `^(\d+\{.[^\}]*\})+$`,
		},
		"testing": {
			withPath: "testing",
			reqPath:  "testing",
			expect:   `^(\d+\{.[^\}]*\})+$`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/%s/?EIO=4&transport=polling", test.reqPath), nil)
			rsp := httptest.NewRecorder()

			svr := socketio.NewServerV3(socketio.WithPath(test.withPath))
			svr.ServeHTTP(rsp, req)

			assert.Regexp(t, test.expect, rsp.Body.String())
		})
	}
}

func TestServerV3(t *testing.T) {
	var opts = []func(*testing.T){}
	var EIOv = 4

	runWithOptions := map[string]testParamsInFn{
		"Polling": func(testDataOpts ...testDataOptFunc) testFn {
			testDataOpts = append(testDataOpts, func(d *testData) { d.version = EIOv }, func(d *testData) { d.transport.value = "polling" })
			return PollingTestV3(opts, testDataOpts...)
		},
		"Websocket": func(testDataOpts ...testDataOptFunc) testFn {
			testDataOpts = append(testDataOpts, func(d *testData) { d.version = EIOv }, func(d *testData) { d.transport.value = "websocket" })
			return WebsocketTestV3(opts, testDataOpts...)
		},
	}

	integration := map[string]testParamsOutFn{
		// spec - https://socket.io/docs/v3/emit-cheatsheet/
		"sending to the client":                                                   SendingToTheClientV3,
		"sending to all clients except sender":                                    SendingToAllClientsExceptTheSenderV3,
		"sending to all clients in 'game' room except sender":                     SendingToAllClientsInGameRoomExceptSenderV3,
		"sending to all clients in 'game1' and/or in 'game2' room, except sender": SendingToAllClientsInGame1AndOrGame2RoomExceptSenderV3,
		"sending to all clients in 'game' room, including sender":                 SendingToAllClientsInGameRoomIncludingSenderV3,
		"sending to all clients in namespace 'myNamespace', including sender":     SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV3,
		"sending to a specific room in a specific namespace, including sender":    SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV3,
		"sending to individual socketid (private message)":                        SendingToIndividualSocketIDPrivateMessageV3,
		"sending with acknowledgement":                                            SendingWithAcknowledgementV3,
		"sending to all connected clients":                                        SendingToAllConnectedClientsV3,

		// extra
		"on event":                                   OnEventV3,
		"reject the client":                          RejectTheClientV3,
		"sending a binary event from the client":     SendingBinaryEventFromClientV3,
		"sending a binary ack event from the client": SendingBinaryAckFromClientV3,
	}

	for name, testParams := range integration {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)...))
		}
	}
}

func PollingTestV3(opts []func(*testing.T), testDataOpts ...testDataOptFunc) testFn {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		var d testData
		for _, testOpt := range testDataOpts {
			testOpt(&d)
		}

		t.Parallel()

		var (
			server = httptest.NewServer(d.server)
			client = make([]*testClient, d.count)
		)

		defer server.Close()

		for i := 0; i < d.count; i++ {
			client[i] = &testClient{polling: &v3PollingClient{
				t:          t,
				d:          d,
				base:       server.URL,
				client:     server.Client(),
				buffer:     new(bytes.Buffer),
				eioVersion: d.version,
			}}

			var queryStr, connStr []string
			if q, ok := d.want["connect_query"]; ok {
				queryStr = q[i]
			}
			if c, ok := d.want["connect"]; ok {
				connStr = c[i]
			}

			client[i].polling.connect(queryStr, connStr)
		}

		// waits for all of the events inside of a onConnection to complete...
		d.syncOn.Wait()

		var x int
		reqSequence := []string{"send1", "grab1", "send2", "grab2"}
		for _, reqType := range reqSequence {
			if request, ok := d.want[reqType]; ok && strings.HasPrefix(reqType, "send") {
				var packetBuf = new(bytes.Buffer)

				for i, packets := range request {
					x++
					if len(packets) == 0 {
						continue
					}

					d.syncOn.Add(1)
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
			if request, ok := d.want[reqType]; ok && strings.HasPrefix(reqType, "grab") {
				for i, want := range request {
					x++
					have := client[i].polling.grab()
					assert.Equal(t, want, have, "[%s] idx: %d", reqType, i)
				}
				continue
			}
		}

		d.syncOn.Wait()

		if xq, ok := d.want["connect_query"]; ok {
			x += len(xq)
		}
		if xc, ok := d.want["connect"]; ok {
			x += len(xc)
		}

		assert.Equal(t, d.count*len(d.want), x) // the wants were actually tested
	}
}

func WebsocketTestV3(opts []func(*testing.T), testDataOpts ...testDataOptFunc) func(*testing.T) {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		var d testData
		for _, testOpt := range testDataOpts {
			testOpt(&d)
		}

		t.Parallel()

		var (
			server = httptest.NewServer(d.server)
			client = make([]*testClient, d.count)
		)

		defer server.Close()

		for i := 0; i < d.count; i++ {
			client[i] = &testClient{websocket: &v3WebsocketClient{
				t:          t,
				d:          d,
				base:       server.URL,
				client:     server.Client(),
				buffer:     new(bytes.Buffer),
				eioVersion: d.version,
			}}

			var queryStr, connStr []string
			if q, ok := d.want["connect_query"]; ok {
				queryStr = q[i]
			}
			if c, ok := d.want["connect"]; ok {
				connStr = c[i]
			}

			client[i].websocket.connect(queryStr, connStr)
		}

		// wait for all onConnection events to complete...
		d.syncOn.Wait()

		var x int
		reqSequence := []string{"send1", "grab1", "send2", "grab2"}
		for _, reqType := range reqSequence {
			if request, ok := d.want[reqType]; ok && strings.HasPrefix(reqType, "send") {

				for i, packets := range request {
					x++
					if len(packets) == 0 {
						continue
					}

					d.syncOn.Add(1)

					for _, packet := range packets {
						time.Sleep(3 * time.Millisecond)
						// check for binary data and send it as binary
						// if it's there
						if len(packet) > 0 && packet[0] == 'b' {
							bin, err := base64.StdEncoding.DecodeString(packet[1:])
							if err != nil {
								panic(err)
							}
							client[i].websocket.send(bytes.NewReader(bin))
							continue
						}
						client[i].websocket.send(strings.NewReader(packet))
					}
				}
				continue
			}
			if request, ok := d.want[reqType]; ok && strings.HasPrefix(reqType, "grab") {
				for i, want := range request {
					x++
					have := client[i].websocket.grab()
					assert.Equal(t, want, have, "[%s] idx: %d", reqType, i)
				}
				continue
			}
		}

		// wait for all emitted events to complete...
		d.syncOn.Wait()

		// check that we hit every "send/grab" that we needed to check...
		if xq, ok := d.want["connect_query"]; ok {
			x += len(xq)
		}
		if xc, ok := d.want["connect"]; ok {
			x += len(xc)
		}
		assert.Equal(t, d.count*len(d.want), x)
	}
}

func SendingToTheClientV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		socket.Emit("hello", str("can you hear me?"), one, serialize.Integer(2), str("abc"))
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToAllClientsExceptTheSenderV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.Broadcast().Emit("broadcast", serialize.String("Hello friends!"))
		}

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToAllClientsInGameRoomExceptSenderV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
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

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToAllClientsInGame1AndOrGame2RoomExceptSenderV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt)%2 == 0 {
			socket.Join("game1")
		}
		if atomic.LoadInt64(&cnt)%3 == 0 {
			socket.Join("game2")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.In("game1").To("game2").Emit("nice game", serialize.String("let's play a game (too)"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToAllClientsInGameRoomIncludingSenderV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt)%2 == 0 {
			socket.Join("game")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v3.To("game").Emit("big-announcement", serialize.String("the game will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToAllClientsInNamespaceMyNamespaceIncludingSenderV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v3.Of("myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v3.Of("myNamespace").OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v3.Of("myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToASpecificRoomInNamespaceMyNamespaceIncludingSenderV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		socket.Join("room")
		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v3.Of("myNamespace").To("room").Emit("event", serialize.String("message"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v3.Of("myNamespace").OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == 0 {
			socket.Join("room")
		}

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			socket.Join("room")
			v3.Of("myNamespace").To("room").Emit("event", serialize.String("message"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToIndividualSocketIDPrivateMessageV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		nextSocketID[(int(atomic.LoadInt64(&cnt)+1) % count)] <- socket.ID().String()

		go func(num int) {
			defer wait.Done()

			socketID := <-nextSocketID[num]
			v3.In(socketID).Emit("hey", serialize.String(fmt.Sprintf("I just met you #%d", num)))
		}(int(atomic.LoadInt64(&cnt)))

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingWithAcknowledgementV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
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

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingToAllConnectedClientsV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		if atomic.LoadInt64(&cnt) == int64(count-1) {
			v3.Emit("*", serialize.String("an event sent to all connected clients"))
		}
		atomic.AddInt64(&cnt, 1)
		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func NameTheTest(name string) socketio.Option {
	return func(v socketio.OptionWith) {
		fmt.Println(">>", fmt.Sprintf("%s: %T", name, v), "<<")
	}
}

func OnEventV3(t *testing.T) []testDataOptFunc {
	o := testingOptionsV3
	o = append(o, NameTheTest("OnEventV3"))

	var (
		v3   = socketio.NewServerV3(o...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
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

		v3.OnDisconnect(func(reason string) {
			defer wait.Done()

			v3.In("room").Emit("say goodbye", serialize.String("disconnecting..."))
		})

		return nil
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func RejectTheClientV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		atomic.AddInt64(&cnt, 1)

		tf := socket.Request().URL.Query().Get("access")
		if tf == "true" {
			socket.Emit("hello", serialize.Integer(1))
			return nil
		}

		return fmt.Errorf("not authorized")
	})

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingBinaryEventFromClientV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.OnConnect(func(socket *socketio.SocketV3) error {
		defer wait.Done()

		atomic.AddInt64(&cnt, 1)
		return nil
	})

	v3.On("hello", testBinaryEventFunc(func(r io.Reader) {
		defer wait.Done()

		have, err := io.ReadAll(r)
		assert.NoError(t, err)
		assert.Equal(t, expect, have)
	}))

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}

func SendingBinaryAckFromClientV3(t *testing.T) []testDataOptFunc {
	var (
		v3   = socketio.NewServerV3(testingOptionsV3...)
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
	v3.Of("/admin").OnConnect(func(socket *socketio.SocketV3) error {
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

	return []testDataOptFunc{
		func(d *testData) { d.server = v3 },
		func(d *testData) { d.count = count },
		func(d *testData) { d.want = want },
		func(d *testData) { d.syncOn = wait },
	}
}
