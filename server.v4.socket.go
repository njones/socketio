package socketio

import (
	"strings"
	"time"

	seri "github.com/njones/socketio/serialize"
	siot "github.com/njones/socketio/transport"
)

var v4ProtectedEventName = map[Event]struct{}{
	"connect":        {},
	"connect_error":  {},
	"disconnect":     {},
	"disconnecting":  {},
	"newListener":    {},
	"removeListener": {},
}

type innTooExceptEmit interface {
	In(...Room) innTooExceptEmit
	To(...Room) innTooExceptEmit
	Except(...Room) innTooExceptEmit
	emit
}

type inSocketV4 struct {
	onConnect map[Namespace]onConnectCallbackVersion4

	prev inSocketV3

	except []Room
}

func (v4 *inSocketV4) clone() inSocketV4 {
	rtn := *v4
	rtn.prev.prev.prev = v4.prev.prev.prev.clone()
	// rtn.onConnect is a map that gets copied by reference
	return rtn
}

func (v4 *inSocketV4) setIsServer(isServer bool)     { v4.prev.setIsServer(isServer) }
func (v4 *inSocketV4) setIsSender(isSender bool)     { v4.prev.setIsSender(isSender) }
func (v4 *inSocketV4) setSocketID(socketID SocketID) { v4.prev.setSocketID(socketID) }
func (v4 *inSocketV4) setPrefix()                    { v4.prev.setPrefix() }
func (v4 *inSocketV4) setNsp(namespace Namespace)    { v4.prev.setNsp(namespace) }
func (v4 *inSocketV4) addID(id siot.SocketID)        { v4.prev.addID(id) }
func (v4 *inSocketV4) addTo(room Room)               { v4.prev.addTo(room) }
func (v4 *inSocketV4) addExcept(room Room) {
	defer v4.prev.prev.prev.l()()
	v4.except = append(v4.except, room)
}

func (v4 inSocketV4) tr() siot.Transporter { return v4.prev.tr() }
func (v4 inSocketV4) nsp() Namespace       { return v4.prev.nsp() }
func (v4 inSocketV4) prefix() string       { return v4.prev.prefix() }
func (v4 inSocketV4) socketID() SocketID   { return v4.prev.socketID() }

func (v4 inSocketV4) OnConnect(callback onConnectCallbackVersion4) {
	v4.onConnect[v4.nsp()] = callback
}
func (v4 inSocketV4) OnDisconnect(callback func(string)) { v4.prev.OnDisconnect(callback) }

func (v4 inSocketV4) On(event Event, callback eventCallback) { v4.prev.On(event, callback) }

// Of - sending to all clients in namespace, including sender
func (v4 inSocketV4) Of(namespace Namespace) inSocketV4 {
	rtn := v4.clone()
	rtn.setNsp(namespace)
	return rtn
}

// In - sending to all clients in room, including sender
func (v4 inSocketV4) In(rooms ...Room) innTooExceptEmit {
	rtn := v4.clone()
	for _, room := range rooms {
		room = strings.Replace(room, v4.prefix(), socketIDPrefix, 1)
		rtn.addTo(room)
	}
	return rtn
}

// To - sending to all clients in room, except sender
func (v4 inSocketV4) To(rooms ...Room) innTooExceptEmit {
	return v4.In(rooms...)
}

// Except - sending to all clients in room, except sender
func (v4 inSocketV4) Except(rooms ...Room) innTooExceptEmit {
	rtn := v4.clone()
	for _, room := range rooms {
		room = strings.Replace(room, v4.prefix(), socketIDPrefix, 1)
		rtn.addExcept(room)
	}
	return rtn
}

// Emit - sending to all connected clients
func (v4 inSocketV4) Emit(event Event, data ...Data) error {
	v3 := v4.prev
	v2 := v3.prev
	v1 := v2.prev

	transport := v1.tr().(siot.Emitter)

	if len(v1.id) == 0 && len(v1.to) == 0 {
		for _, id := range transport.Sockets(v1.nsp()).IDs() {
			if id == v1._socketID && v1.isSender {
				continue // skip sending back to sender
			}
			v1.addID(id)
		}
		// send to local server ... since this is not a broadcast
		if ns, ok := v1.events[v1.nsp()]; ok {
			if events, ok := ns[event][v1._socketID]; ok {
				events.Callback(seri.Convert(data).ToInterface()...)
			}
		}
		return v1.emit(event, data...)
	}

	var uniqueID = map[SocketID]struct{}{}
	for _, exceptRoom := range v4.except {
		rooms, err := transport.Sockets(v1.nsp()).FromRoom(exceptRoom)
		if err != nil {
			ErrFromRoomFailed.F(err)
		}
		for _, id := range rooms {
			uniqueID[id] = struct{}{}
		}
	}
	for _, toRoom := range v1.to {
		ids, err := transport.Sockets(v1.nsp()).FromRoom(toRoom)
		if err != nil {
			ErrFromRoomFailed.F(err)
		}

		for _, id := range ids {
			if id == v1._socketID && !v1.isServer {
				continue // skip sending back to sender
			}

			if _, inSet := uniqueID[id]; !inSet {
				v1.addID(id)
				uniqueID[id] = struct{}{}
			}
		}
	}

	return v1.emit(event, data...)
}

type onConnectCallbackVersion4 = func(*SocketV4) error

type SocketV4 struct {
	inSocketV4

	han handshakeV4
	req *Request
}

func (v4 *SocketV4) ID() SocketID           { return SocketID(v4.prefix()) + v4.socketID() }
func (v4 *SocketV4) Request() *Request      { return v4.req }
func (v4 *SocketV4) Handshake() handshakeV4 { v4.han.init(); return v4.han }

func (v4 *SocketV4) Emit(event Event, data ...Data) error {
	v4.addID(v4.socketID())
	return v4.prev.Emit(event, data...)
}

func (v4 *SocketV4) Join(room Room) error {
	return v4.tr().Join(v4.nsp(), v4.socketID(), strings.Replace(room, v4.prefix(), socketIDPrefix, 1))
}
func (v4 *SocketV4) Leave(room Room) error {
	return v4.tr().Leave(v4.nsp(), v4.socketID(), room)
}

func (v4 *SocketV4) Broadcast() emit                { v4.setIsSender(true); return v4.inSocketV4 }
func (v4 *SocketV4) Volatile() emit                 { return v4 } // NOT IMPLEMENTED...
func (v4 *SocketV4) Compress(compress bool) emit    { return v4 } // NOT IMPLEMENTED...
func (v4 *SocketV4) Timeout(dur time.Duration) emit { return v4 } // NOT IMPLEMENTED...
