module github.com/njones/socketio/engineio

go 1.17

require github.com/njones/socketio/engineio/session v0.0.0

require github.com/njones/socketio/engineio/protocol v0.0.0

require github.com/njones/socketio/engineio/transport v0.0.0

require (
	github.com/klauspost/compress v1.10.3 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

replace github.com/njones/socketio/engineio/session => ./session

replace github.com/njones/socketio/engineio/protocol => ./protocol

replace github.com/njones/socketio/engineio/transport => ./transport
