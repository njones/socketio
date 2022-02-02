module github.com/njones/socketio

go 1.17

require github.com/njones/socketio/adaptor/transport/map v0.0.0

require github.com/njones/socketio/engineio/transport v0.0.0 // indirect

require github.com/njones/socketio/engineio/session v0.0.0 // indirect

require (
	github.com/klauspost/compress v1.10.3 // indirect
	github.com/njones/socketio/engineio/protocol v0.0.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

require github.com/njones/socketio/transport v0.0.0

require github.com/njones/socketio/protocol v0.0.0

require github.com/njones/socketio/engineio v0.0.0

require github.com/njones/socketio/session v0.0.0

replace github.com/njones/socketio/engineio => ./engineio

replace github.com/njones/socketio/adaptor/transport/map => ./adaptor/transport/map

replace github.com/njones/socketio/engineio/transport => ./engineio/transport

replace github.com/njones/socketio/engineio/session => ./engineio/session

replace github.com/njones/socketio/engineio/protocol => ./engineio/protocol

replace github.com/njones/socketio/transport => ./transport

replace github.com/njones/socketio/protocol => ./protocol

replace github.com/njones/socketio/session => ./session
