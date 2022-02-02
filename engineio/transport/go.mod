module github.com/njones/socketio/engineio/transport

go 1.17

require github.com/njones/socketio/engineio/protocol v0.0.0

require (
	github.com/gobwas/ws v1.0.2
	github.com/njones/socketio/engineio/session v0.0.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	nhooyr.io/websocket v1.8.7
)

require (
	github.com/gobwas/httphead v0.0.0-20180130184737-2c6c146eadee // indirect
	github.com/gobwas/pool v0.2.0 // indirect
	github.com/klauspost/compress v1.10.3 // indirect
	golang.org/x/sys v0.0.0-20200116001909-b77594299b42 // indirect
)

replace github.com/njones/socketio/engineio/protocol => ../protocol

replace github.com/njones/socketio/engineio/session => ../session
