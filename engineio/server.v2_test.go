package engineio_test

import (
	"fmt"
	"testing"

	"github.com/njones/socketio"
	"github.com/njones/socketio/engineio"
)

// CORS
// XHR2

type (
	testFn          func(*testing.T)
	testParamsInFn  func(engineio.Server) testFn
	testParamsOutFn func(*testing.T) engineio.Server
)

func TestServerV2(t *testing.T) {
	var opts = []func(*testing.T){}
	var EIOv = 3

	runWithOptions := map[string]testParamsInFn{
		"cors": func(v2 engineio.Server) testFn {
			return CORSTestV2(opts, EIOv, v2)
		},
	}

	tests := map[string]testParamsOutFn{
		"sending to the client": SendingToTheClientV2,
	}

	for name, testParams := range tests {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}

}

func CORSTestV2(opts []func(*testing.T), EIOv int, v2 engineio.Server) testFn {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}
	}
}

func SendingToTheClientV2(t *testing.T) (a socketio.Server) {
	// 	var (
	// 		v2   = socketio.NewServerV2(testingQuickPoll)
	// 		wait = new(sync.WaitGroup)

	// 		want = map[string][][]string{
	// 			"grab1": {
	// 				{`42["hello","can you hear me?",1,2,"abc"]`},
	// 				{`42["hello","can you hear me?",1,2,"abc"]`},
	// 				{`42["hello","can you hear me?",1,2,"abc"]`},
	// 			},
	// 		}
	// 		count = len(want["grab1"])

	// 		str = serialize.String
	// 		one = serialize.Int(1)
	// 	)

	// 	wait.Add(count)
	// 	v2.OnConnect(func(socket *socketio.SocketV2) error {
	// 		defer wait.Done()

	// 		socket.Emit("hello", str("can you hear me?"), one, serialize.Int(2), str("abc"))
	// 		return nil
	// 	})

	// 	return v2, count, want, wait
	return
}
