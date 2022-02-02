module github.com/njones/socketio/transport

go 1.17

require (
	github.com/njones/socketio/engineio/protocol v0.0.0
	github.com/njones/socketio/engineio/transport v0.0.0
	github.com/njones/socketio/protocol v0.0.0
	github.com/njones/socketio/session v0.0.0
)

require github.com/njones/socketio/engineio/session v0.0.0 // indirect

replace (
	github.com/njones/socketio/engineio/protocol => ../engineio/protocol
	github.com/njones/socketio/engineio/session => ../engineio/session
	github.com/njones/socketio/engineio/transport => ../engineio/transport
	github.com/njones/socketio/protocol => ../protocol
	github.com/njones/socketio/session => ../session

)
