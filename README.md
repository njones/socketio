# SocketIO [![GoDoc](https://pkg.go.dev/badge/github.com/njones/socketio?utm_source=godoc)](https://pkg.go.dev/github.com/njones/socketio) 

This Go language SocketIO library aims to support all past, current and future versions of the Socket.io (and Engine.io) protocols and servers.

The library currently supports the following versions:

| SocketIO protocols | EngineIO protocols | SocketIO Server | EngineIO Server |
|--------------------|--------------------|-----------------|-----------------|
| v1 (unspecified)   | v2                 | v2.4.1          | v1.8.x          |
| v2                 | v3                 | v3.0.1          | v2.1.x          |
| v3                 | v4                 |                 | v3.5.x          |
| v4                 | v5                 |                 |                 |
|                    | v6                 |                 |                 |

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
```

_inside of a function_

```go
    // use the latest SocketIO (v4) and EngineIO (v4) version 
    // setting the EngineIO ping interval to 10 seconds
    server := sio.NewServer(eio.WithPingInterval(10 * time.Second))

    // use a OnConnect handler for incoming "connection" messages
    server.OnConnect(func(socket *socketio.SocketV4) error {

        // add serializable data to variables
        // See: https://github.com/njones/socketio/blob/main/serialize.go for standard serialized types.
        // Custom types can be serialized with the following interface: 
        //   https://github.com/njones/socketio/blob/cdf59a60d92c70862c859ade8415f7399e8fea37/serialize.go#L12
        canYouHear := sio.String("can you hear me?")
        
        var extra sio.String = "abc"
        var questions sio.Int = 1
        var responses = sio.Int(2)

        // send out a message to the hello 
        socket.Emit("hello", canYouHear, question, responses, extra)

        return nil
    })

```

## TODO

The following is in no particular order. Please open an Issue for priority or open a PR to contribute to this list.

- [ ] Develop a Client 
- [ ] Develop a Redis Transport
- [ ] Makefile for all individual version builds
- [ ] Makefile for all individual version git commits
- [ ] Document all public functions
- [ ] Complete SocketIO Version 4
- [ ] Complete EIO Server Version 5
- [ ] Complete EIO Server Version 6
- [ ] Flesh out all tests
- [ ] Documentation
- [ ] Complete this README

## License

The MIT license. 

_See the LICENSE file for more information_
