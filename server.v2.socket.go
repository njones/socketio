package socketio

import (
	"strings"

	siot "github.com/njones/socketio/transport"
)

type inSocketV2 struct {
	onConnect map[Namespace]onConnectCallbackVersion2

	prev inSocketV1
}

func (v2 *inSocketV2) clone() inSocketV2 {
	rtn := *v2
	rtn.prev = v2.prev.clone()
	// rtn.onConnect is a map that gets copied by reference
	return rtn
}

func (v2 *inSocketV2) setIsServer(isServer bool)     { v2.prev.setIsServer(isServer) }
func (v2 *inSocketV2) setIsSender(isSender bool)     { v2.prev.setIsSender(isSender) }
func (v2 *inSocketV2) setSocketID(socketID SocketID) { v2.prev.setSocketID(socketID) }
func (v2 *inSocketV2) setPrefix()                    { v2.prev.setPrefix() }
func (v2 *inSocketV2) setNsp(namespace Namespace)    { v2.prev.setNsp(namespace) }
func (v2 *inSocketV2) addID(id siot.SocketID)        { v2.prev.addID(id) }
func (v2 *inSocketV2) addTo(room Room)               { v2.prev.addTo(room) }

func (v2 inSocketV2) tr() siot.Transporter { v1 := v2.prev; return v1.tr() }
func (v2 inSocketV2) nsp() Namespace       { return v2.prev.nsp() }
func (v2 inSocketV2) prefix() string       { return v2.prev.prefix() }
func (v2 inSocketV2) socketID() SocketID   { return v2.prev.socketID() }

func (v2 inSocketV2) OnConnect(callback onConnectCallbackVersion2) {
	v2.onConnect[v2.nsp()] = callback
}
func (v2 inSocketV2) OnDisconnect(callback func(string))     { v2.prev.OnDisconnect(callback) }
func (v2 inSocketV2) On(event Event, callback eventCallback) { v2.prev.On(event, callback) }

func (v2 inSocketV2) Of(namespace Namespace) inSocketV2 {
	rtn := v2.clone()
	rtn.setNsp(namespace)
	return rtn
}

func (v2 inSocketV2) In(room Room) inToEmit {
	room = strings.Replace(room, v2.prefix(), socketIDPrefix, 1)
	rtn := v2.clone()
	rtn.addTo(room)
	return rtn
}

func (v2 inSocketV2) To(room Room) inToEmit { return v2.In(room) }

func (v2 inSocketV2) Emit(event Event, data ...Data) error {
	return v2.prev.Emit(event, data...)
}

type onConnectCallbackVersion2 = func(*SocketV2) error

type SocketV2 struct {
	inSocketV2

	req *Request
}

func (v2 *SocketV2) ID() SocketID      { return SocketID(v2.prefix()) + v2.socketID() }
func (v2 *SocketV2) Request() *Request { return v2.req }

func (v2 *SocketV2) Emit(event Event, data ...Data) error {
	v2.addID(v2.socketID())
	return v2.prev.emit(event, data...)
}

func (v2 *SocketV2) Join(room Room) error {
	room = strings.Replace(room, v2.prefix(), socketIDPrefix, 1)
	return v2.tr().Join(v2.nsp(), v2.socketID(), room)
}
func (v2 *SocketV2) Leave(room Room) error {
	return v2.tr().Leave(v2.nsp(), v2.socketID(), room)
}

func (v2 *SocketV2) Broadcast() emit             { v2.setIsSender(true); return v2.inSocketV2 }
func (v2 *SocketV2) Volatile() emit              { return v2 } // NOT IMPLEMENTED...
func (v2 *SocketV2) Compress(compress bool) emit { return v2 } // NOT IMPLEMENTED...
func (v2 *SocketV2) Binary(binary bool) emit     { return v2 } // NOT IMPLEMENTED...
