package socketio_test

import (
	"encoding/hex"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/njones/socketio"
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

var onConnectSocketEmit = func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	v1 := socketio.NewServerV1()

	callbacksComplete.Add(1)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer callbacksComplete.Done()

		socket.Emit("hello", socketio.String("can you hear me?"), socketio.Int(1), socketio.Int(2), socketio.String("abc"))
		return nil
	})
	return v1
}

var onConnectBroadcastEmit = func(t *testing.T, callbacksComplete *sync.WaitGroup) socketio.Server {
	v1 := socketio.NewServerV1()

	callbacksComplete.Add(1)
	v1.OnConnect(func(socket *socketio.SocketV1) error {
		defer callbacksComplete.Done()

		socket.Broadcast().Emit("hello", socketio.String("hello friends!"))
		return nil
	})
	return v1
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
				socket.To("game").Emit("nice game", socketio.String("let's play a game"))
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
				socket.To("game1").To("game2").Emit("nice game", socketio.String("let's play a game (too)"))
			}

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
		err := socket.Emit("question", socketio.String("do you think so?"), socketio.CallbackWrap{
			Parameters: []socketio.Serializable{socketio.Str, socketio.Str},
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
				socket.To("game").Emit("nice game", socketio.String("let's play a game"))
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

	v1.On("hello", socketio.CallbackWrap{
		Parameters: []socketio.Serializable{socketio.Bin},
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

// TestServerV1Basic tests all of the basic functions of the server
func TestServerV1Basic(t *testing.T) {

	var tests = []struct {
		name             string
		numSvrs          int
		eioVer, tspType  string
		server           func(*testing.T, *sync.WaitGroup) socketio.Server // the function that holds what we are testing
		incomingMessages []mesg
	}{
		{
			name:   "onConnect",
			server: onConnect,
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=1", data: "2:40"},
			},
		},
		{
			name:   "onConnect with socket.Emit",
			server: onConnectSocketEmit,
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=2", data: "2:40"},
				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=3", data: `40:42["hello","can you hear me?",1,2,"abc"]`},
			},
		},
		{
			name:    "broadcast.Emit",
			numSvrs: 3,
			server:  onConnectBroadcastEmit,
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}&a=4", data: "2:40"},
				{method: "GET", url: "/socket.io/?${eio}&${sid_0}&${t}&a=5", data: ``},
				{method: "GET", url: "/socket.io/?${eio}&${sid_1}&${t}&a=6", data: `28:42["hello","hello friends!"]`},
				{method: "GET", url: "/socket.io/?${eio}&${sid_2}&${t}&a=7", data: `28:42["hello","hello friends!"]`},
			},
		},
		{
			name:    "room.Emit",
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
		{
			name:    "room.DualEmit",
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
		{
			name:   "socket.Emit with Ack",
			server: onConnectEmitAckDefaultWrap,
			incomingMessages: []mesg{
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "2:40"},
				{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: `27:43,1["answer", "answer123"]`},
			},
		},
		{
			name:    "room.Leave",
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
		// {
		// 	name:   "socket.Event with Binary",
		// 	server: onConnectEmitAckDefaultWrap,
		// 	incomingMessages: []mesg{
		// 		{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: `42:51-["hello",{"_placeholder":true,"num":0}]`},
		// 		{method: "POST", url: "/socket.io/?${eio}&${sid_0}&${t}", data: "\x61\x6e\x73\x77\x65\x72\x20\x31\x32\x33"},
		// 	},
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t2 *testing.T) {
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

/*

The full emit cheatsheet
AckID from callback function

*/
