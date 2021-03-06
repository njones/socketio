package socketio_test

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/njones/socketio"
	"github.com/njones/socketio/callback"
	"github.com/njones/socketio/serialize"
	"github.com/stretchr/testify/assert"
)

var onConnect = func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	v1 := socketio.NewServerV1()

	callbacksComplete.Add(1)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer callbacksComplete.Done()

		return nil
	})
	return v1
}

var onConnectSocketEmit = func(numCallbacks int) func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			if cntCallbacks == (numCallbacks - 1) {
				// send back to all client that just connected
				socket.Emit("hello", serialize.String("can you hear me?"), serialize.Int(1), serialize.Int(2), serialize.String("abc"))
			}

			cntCallbacks++
			return nil
		})
		return v1
	}
}

var onConnectBroadcastEmit = func(numCallbacks int) func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			if cntCallbacks == (numCallbacks - 1) {
				// send to all clients except the sender
				socket.Broadcast().Emit("hello", serialize.String("hello friends!"))
			}

			cntCallbacks++
			return nil
		})
		return v1
	}
}

var onConnectServerEmit = func(numCallbacks int) func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			if cntCallbacks == (numCallbacks - 1) {
				// send back to all clients including the sender
				v1.Emit("hello", serialize.String("can you hear me?"), serialize.Int(1), serialize.Int(2), serialize.String("abc"))
			}

			cntCallbacks++
			return nil
		})
		return v1
	}
}

var onConnectRoomEmit = func(numCallbacks int) func(*testing.T, *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			if cntCallbacks > 0 {
				socket.Join("game")
			}

			if cntCallbacks == (numCallbacks - 1) {
				socket.To("game").Emit("nice game", serialize.String("let's play a game"))
			}

			cntCallbacks++
			return nil
		})

		return v1
	}
}

var onConnectDualRoomEmit = func(numCallbacks int) func(*testing.T, *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			if cntCallbacks%2 == 0 {
				socket.Join("game1")
			}

			if cntCallbacks%3 == 0 {
				socket.Join("game2")
			}

			if cntCallbacks == (numCallbacks - 1) {
				socket.Join("game1")
				socket.Join("game2")
				socket.To("game1").To("game2").Emit("nice game", serialize.String("let's play a game (too)"))
			}

			cntCallbacks++
			return nil
		})

		return v1
	}
}

var onConnectAllInEmit = func(numCallbacks int) func(*testing.T, *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			if cntCallbacks%2 == 0 {
				socket.Join("game1")
			}

			if cntCallbacks%3 == 0 {
				socket.Join("game2")
			}

			if cntCallbacks == (numCallbacks - 1) {
				socket.Join("game1")
				socket.Join("game2")
				v1.In("game1").In("game2").Emit("nice game", serialize.String("let's play a game (too)"))
			}

			cntCallbacks++
			return nil
		})

		return v1
	}
}

var onConnectServerNamespaceEmit = func(numCallbacks int) func(*testing.T, *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			log.Println(">> '/'", cntCallbacks)
			cntCallbacks++
			return nil
		})
		v1.Of("/myNamespace").OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			log.Println(">> 'myNamespace'", cntCallbacks)

			if cntCallbacks == (numCallbacks - 1) {
				v1.Of("/myNamespace").Emit("bigger-announcement", serialize.String("the tournament will start soon"))
			}

			cntCallbacks++
			return nil
		})

		return v1
	}
}

var onConnectServerToSocketIDEmit = func(numCallbacks int) func(*testing.T, *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		nextSocketID := make([]chan string, numCallbacks)
		for i := range nextSocketID {
			nextSocketID[i] = make(chan string, 1)
		}

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			nextSocketID[(cntCallbacks+1)%numCallbacks] <- string(socket.ID)

			go func(num int) {
				defer callbacksComplete.Done()

				socketID := <-nextSocketID[num]
				v1.To(socketID).Emit("hey", serialize.String(fmt.Sprintf("I just met you #%d", num)))
			}(cntCallbacks)

			cntCallbacks++
			return nil
		})

		return v1
	}
}

var onConnectEmitAckDefaultWrap = func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	v1 := socketio.NewServerV1()

	callbacksComplete.Add(1)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer callbacksComplete.Done()

		callbacksComplete.Add(1)
		err := socket.Emit("question", serialize.String("do you think so?"), callback.Wrap{
			Parameters: []serialize.Serializable{serialize.Str, serialize.Str},
			Func: func() interface{} {
				return func(val, oop string) error {
					callbacksComplete.Done()

					assert.Equal(t, "answer", val)
					assert.Equal(t, "answer123", oop)

					return nil
				}
			},
		})

		assert.NoError(t, err)

		return nil
	})

	return v1
}

var onConnectRoomLeave = func(numCallbacks int) func(*testing.T, *sync.WaitGroup) socketio.Server {
	var cntCallbacks int
	return func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
		v1 := socketio.NewServerV1()

		callbacksComplete.Add(numCallbacks)
		v1.OnConnect(func(socket *socketio.SocketV1) error {
			defer callbacksComplete.Done()

			socket.Join("game")

			if cntCallbacks == 1 {
				socket.Leave("game")
			}

			if cntCallbacks == (numCallbacks - 1) {
				socket.To("game").Emit("nice game", serialize.String("let's play a game"))
			}

			cntCallbacks++
			return nil
		})

		return v1
	}
}

var onConnectEmitAckDefaultWrapWithBin = func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	v1 := socketio.NewServerV1()

	callbacksComplete.Add(1)

	v1.On("hello", callback.Wrap{
		Parameters: []serialize.Serializable{serialize.Bin},
		Func: func() interface{} {
			return func(data io.Reader) error {
				callbacksComplete.Done()

				b, err := io.ReadAll(data)
				if assert.NoError(t, err) {
					return err
				}

				have := strings.ToLower(hex.EncodeToString(b))
				want := "616e7377657220313233"

				assert.Equal(t, want, have)

				return nil
			}
		},
	})

	return v1
}

/*
{
	name:   "socket.Event with Binary",
	server: onConnectEmitAckDefaultWrap,
	incomingMessages: []mesg{
		{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: `43:451-["hello",{"_placeholder":true,"num":0}]`},
		{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "\x61\x6e\x73\x77\x65\x72\x20\x31\x32\x33"},
	},
},
*/

// TestServerV1Basic tests all of the basic functions of the server
func TestServerV1Basic(t *testing.T) {

	runOnly := map[string]struct{}{"server.ToSocketIDEmit": {}}

	var tests = map[string]struct {
		numSvrs          int
		eioVer, tspType  string
		server           func(*testing.T, *sync.WaitGroup) socketio.Server // the function that holds what we are testing
		incomingMessages []mesg
	}{
		"onConnect": {
			server: onConnect,
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=0", data: "2:40"},
			},
		},
		"socket.Emit": {
			numSvrs: 3,
			server:  onConnectSocketEmit(3),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=1", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}&a=2", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}&a=3", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=4", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}&a=5", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}&a=6", data: `40:42["hello","can you hear me?",1,2,"abc"]`},
			},
		},
		"broadcast.Emit": {
			numSvrs: 3,
			server:  onConnectBroadcastEmit(3),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=7", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}&a=8", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}&a=9", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=A", data: `28:42["hello","hello friends!"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}&a=B", data: `28:42["hello","hello friends!"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}&a=C", data: ``},
			},
		},
		"server.Emit": {
			numSvrs: 3,
			server:  onConnectServerEmit(3),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=D", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}&a=E", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}&a=F", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=0", data: `40:42["hello","can you hear me?",1,2,"abc"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}&a=1", data: `40:42["hello","can you hear me?",1,2,"abc"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}&a=2", data: `40:42["hello","can you hear me?",1,2,"abc"]`},
			},
		},
		"server-namespace.Emit": {
			numSvrs: 6,
			server:  onConnectServerNamespaceEmit(6),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=AA", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}&a=BB", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}&a=CC", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_3}&${t}&a=DD", data: "14:40/myNamespace"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_4}&${t}&a=EE", data: "14:40/myNamespace"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_5}&${t}&a=FF", data: "14:40/myNamespace"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=aa", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}&a=bb", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}&a=cc", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_3}&${t}&a=dd", data: `71:42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_4}&${t}&a=ee", data: `71:42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_5}&${t}&a=ff", data: `71:42/myNamespace,["bigger-announcement","the tournament will start soon"]`},
			},
		},
		"server.ToSocketIDEmit": {
			numSvrs: 3,
			server:  onConnectServerToSocketIDEmit(3),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=0", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}&a=1", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}&a=2", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=a", data: `29:42["hey","I just met you #1"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}&a=b", data: `29:42["hey","I just met you #2"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}&a=c", data: `29:42["hey","I just met you #0"]`},
			},
		},
		"room.Emit": {
			numSvrs: 4,
			server:  onConnectRoomEmit(4),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_3}&${t}", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}", data: `35:42["nice game","let's play a game"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}", data: `35:42["nice game","let's play a game"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_3}&${t}", data: ``},
			},
		},
		"room.DualEmit": {
			numSvrs: 8,
			server:  onConnectDualRoomEmit(8),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_3}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_4}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_5}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_6}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_7}&${t}", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}_de", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_3}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_4}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_5}&${t}_de&break=true", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_6}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_7}&${t}_de", data: ``},
			},
		},
		"room.AllInEmit": {
			numSvrs: 8,
			server:  onConnectAllInEmit(8),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_3}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_4}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_5}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_6}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_7}&${t}", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}_de", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_3}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_4}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_5}&${t}_de&break=true", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_6}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_7}&${t}_de", data: `41:42["nice game","let's play a game (too)"]`},
			},
		},
		"socket.Emit with Ack": {
			server: onConnectEmitAckDefaultWrap,
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: `27:43,1["answer", "answer123"]`},
			},
		},
		"room.Leave": {
			numSvrs: 4,
			server:  onConnectRoomLeave(4),
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_1}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_2}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_3}&${t}", data: "2:40"},

				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}", data: `35:42["nice game","let's play a game"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}", data: `35:42["nice game","let's play a game"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_3}&${t}", data: ``},
			},
		},
	}

	for name, test := range tests {

		if len(runOnly) > 0 {
			if _, ok := runOnly[name]; !ok {
				continue
			}
		}

		t.Run(name, func(t2 *testing.T) {
			callbacksComplete := new(sync.WaitGroup)
			hdlr := test.server(t2, callbacksComplete)
			svr := httptest.NewServer(hdlr)

			ver := test.eioVer
			if ver == "" {
				ver = "2" // the default
			}
			typ := test.tspType
			if typ == "" {
				typ = "polling" // the defaut
			}

			sid := connect(t2, svr, test.numSvrs, ver, typ)
			for _, msg := range test.incomingMessages {
				testMessage(t2, svr, ver, sid, msg)
			}
			disconnect(t2, svr, ver, sid)

			callbacksComplete.Wait()
			svr.Close()
		})
	}
}
