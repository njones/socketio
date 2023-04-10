# SocketIO [![GoDoc](https://pkg.go.dev/badge/github.com/njones/socketio?utm_source=godoc)](https://pkg.go.dev/github.com/njones/socketio) 

This Go language SocketIO library aims to support all past, current and future versions of the Socket.io (and Engine.io) protocols and servers.

The library currently supports the following versions:

| SocketIO protocol | EngineIO protocol/payload | SocketIO Server | EngineIO Server     |
|-------------------|---------------------------|-----------------|---------------------|
| v1 (unspecified)  | (unspecified)             | v1.0.x          | (unspecified)       |
| v2                | v2 / v2                   | v2.4.x          | v2.1.x              |
| v3                | v3 / v3                   | v3.0.x          | v3.6.x              |
| v4                | v4 / v4                   | v4.5.x          | v4.1.x              |
| v5                |                           |                 | v5.2.x              |
|                   |                           |                 | v6.x (same as v5.x) |

Getting the correct features/protocols/versions included inside which SocketIO and EngineIO Server versions can be confusing at times, therefore some servers may initially be implemented incorrectly, or not have features implemented. Please open a ticket for any discrepancies. 

This library is very new and **we're looking for beta testers.**

## Contents

- [Install](#install)
- [Example](#example)
- [TODO List](#todo)
- [License](#license)

## Install

```bash
go get github.com/njones/socketio
```

## Example

### A simple example: sending a message out when a client connects

```go
import (
	"log"
	"net/http"
	"time"

	sio "github.com/njones/socketio"
	eio "github.com/njones/socketio/engineio"
	eiot "github.com/njones/socketio/engineio/transport"
	ser "github.com/njones/socketio/serialize"
)

func main() {
	port := ":3000"

	server := sio.NewServer(
		eio.WithPingInterval(300*1*time.Millisecond),
		eio.WithPingTimeout(200*1*time.Millisecond),
		eio.WithMaxPayload(1000000),
		eio.WithTransportOption(eiot.WithGovernor(1500*time.Microsecond, 500*time.Microsecond)),
	)

	// use a OnConnect handler for incoming "connection" messages
	server.OnConnect(func(socket *sio.SocketV4) error {

		canYouHear := ser.String("can you hear me?")
		extra := ser.String("abc")

		var questions = ser.Integer(1)
		var responses = ser.Map(map[string]interface{}{"one": "no"})

		// send out a message to the hello
		socket.Emit("hello", canYouHear, questions, responses, extra)

		return nil
	})

	log.Printf("serving port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, server))
}
```

### A more complicated example: emitting and listening to a custom event
```go
import (
	"log"
	"net/http"
	"time"

	sio "github.com/njones/socketio"
	eio "github.com/njones/socketio/engineio"
	eiot "github.com/njones/socketio/engineio/transport"
	ser "github.com/njones/socketio/serialize"
)

// Define a custom wrapper 
type CustomWrap func(string, string) error

// Define your callback
func (cc CustomWrap) Callback(data ...interface{}) error {
	a, aOK := data[0].(string)
	b, bOK := data[1].(string)

	if !aOK || !bOK {
		return fmt.Errorf("bad parameters")
	}

	return cc(a, b)
}

func main() {
	port := ":3000"
    server := socketio.NewServer()
    server.OnConnect(func(socket *sio.SocketV4) error {
         // Implement your callback for a custom event
         socket.On("myEvent", CustomWrap(func(a string, b string) error{
            socket.emit("hello", a, b)
            return nil
         })
    }
	log.Printf("serving port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, server))
}
```

## TODO

The following is in no particular order. Please open an Issue for priority or open a PR to contribute to this list.

- [x] Flesh out all tests
- [ ] Document all public functions
- [ ] Documentation
- [ ] Develop a Client 
- [ ] Develop a Redis Transport
- [ ] Makefile for all individual version builds
- [ ] Makefile for all individual version git commits
- [x] Complete SocketIO Version 4
- [x] Complete EIO Server Version 5
- [x] Complete EIO Server Version 6
- [ ] Complete this README

## License

The MIT license. 

_See the LICENSE file for more information_
