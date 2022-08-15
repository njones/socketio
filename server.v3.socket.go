package socketio

import (
	"strings"

	siot "github.com/njones/socketio/transport"
)

var v3ProtectedEventName = map[Event]struct{}{
	"connect":        {},
	"connect_error":  {},
	"disconnect":     {},
	"disconnecting":  {},
	"newListener":    {},
	"removeListener": {},
}

type inSocketV3 struct {
	onConnect map[Namespace]onConnectCallbackVersion3

	prev inSocketV2
}

func (v3 *inSocketV3) clone() inSocketV3 {
	rtn := *v3
	rtn.prev.prev = v3.prev.prev.clone()
	// rtn.onConnect is a map that gets copied by reference
	return rtn
}

func (v3 *inSocketV3) setIsServer(isServer bool)     { v3.prev.setIsServer(isServer) }
func (v3 *inSocketV3) setIsSender(isSender bool)     { v3.prev.setIsSender(isSender) }
func (v3 *inSocketV3) setSocketID(socketID SocketID) { v3.prev.setSocketID(socketID) }
func (v3 *inSocketV3) setPrefix()                    { v3.prev.setPrefix() }
func (v3 *inSocketV3) setNsp(namespace Namespace)    { v3.prev.setNsp(namespace) }
func (v3 *inSocketV3) addID(id siot.SocketID)        { v3.prev.addID(id) }
func (v3 *inSocketV3) addTo(room Room)               { v3.prev.addTo(room) }

func (v3 inSocketV3) tr() siot.Transporter { return v3.prev.tr() }
func (v3 inSocketV3) nsp() Namespace       { return v3.prev.nsp() }
func (v3 inSocketV3) prefix() string       { return v3.prev.prefix() }
func (v3 inSocketV3) socketID() SocketID   { return v3.prev.socketID() }

func (v3 inSocketV3) OnConnect(callback onConnectCallbackVersion3) {
	v3.onConnect[v3.nsp()] = callback
}
func (v3 inSocketV3) OnDisconnect(callback func(string)) { v3.prev.OnDisconnect(callback) }

func (v3 inSocketV3) On(event Event, callback eventCallback) { v3.prev.On(event, callback) }

// Of - sending to all clients in namespace, including sender
func (v3 inSocketV3) Of(namespace Namespace) inSocketV3 {
	rtn := v3.clone()
	rtn.setNsp(namespace)
	return rtn
}

// In - sending to all clients in room, including sender
func (v3 inSocketV3) In(room Room) inToEmit { return v3.To(room) }

// To - sending to all clients in room, except sender
func (v3 inSocketV3) To(room Room) inToEmit {
	room = strings.Replace(room, v3.prefix(), socketIDPrefix, 1)
	rtn := v3.clone()
	rtn.addTo(room)
	return rtn
}

// Emit - sending to all connected clients
func (v3 inSocketV3) Emit(event Event, data ...Data) error {
	return v3.prev.Emit(event, data...)
}

type onConnectCallbackVersion3 = func(*SocketV3) error

type SocketV3 struct {
	inSocketV3

	req *Request
}

func (v3 *SocketV3) ID() SocketID      { return SocketID(v3.prefix()) + v3.socketID() }
func (v3 *SocketV3) Request() *Request { return v3.req }

func (v3 *SocketV3) Emit(event Event, data ...Data) error {
	v3.addID(v3.socketID())
	return v3.prev.Emit(event, data...)
}

func (v3 *SocketV3) Join(room Room) error {
	return v3.tr().Join(v3.nsp(), v3.socketID(), strings.Replace(room, v3.prefix(), socketIDPrefix, 1))
}
func (v3 *SocketV3) Leave(room Room) error {
	return v3.tr().Leave(v3.nsp(), v3.socketID(), room)
}

func (v3 *SocketV3) Broadcast() emit             { v3.setIsSender(true); return v3.inSocketV3 }
func (v3 *SocketV3) Volatile() emit              { return v3 } // NOT IMPLEMENTED...
func (v3 *SocketV3) Compress(compress bool) emit { return v3 } // NOT IMPLEMENTED...
