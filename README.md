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
    import sio github.com/njones/socketio
    import eio github.com/njones/socketio/engineio
    import ser github.com/njones/socketio/serialize
```

_inside of a function_

```go
    // use the latest SocketIO (v4) and EngineIO (v4) version 
    // setting the EngineIO ping interval to 10 seconds
    server := sio.NewServer(eio.WithPingInterval(10 * time.Second))

    // use a OnConnect handler for incoming "connection" messages
    server.OnConnect(func(socket *socketio.SocketV4) error {

        // add serializable data to variables
        // See: https://github.com/njones/socketio/blob/main/serialize/serialize.go for standard serialized types.
        // Custom types can be serialized with the following interface: 
        // https://github.com/njones/socketio/blob/7c6c70708442f9e8d4b33991389d9c6d155da699/serialize/serialize.go#L12
        canYouHear := sio.String("can you hear me?")
        
        var questions ser.Int = 1
        var responses = ser.Int(2)
        var extra ser.String = "abc"

        // send out a message to the hello 
        socket.Emit("hello", canYouHear, question, responses, extra)

        return nil
    })

```

## TODO

The following is in no particular order. Please open an Issue for priority or open a PR to contribute to this list.

- [ ] Flesh out all tests
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
