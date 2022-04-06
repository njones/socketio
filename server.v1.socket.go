package socketio

import (
	"fmt"
	"io"

	cabk "github.com/njones/socketio/callback"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

type onConnectCallbackVersion1 = func(*SocketV1) error

var v1ProtectedEventName = map[Event]struct{}{
	"connect":           {},
	"connection":        {},
	"connect_error":     {},
	"connect_timeout":   {},
	"error":             {},
	"disconnect":        {},
	"disconnecting":     {},
	"newListener":       {},
	"reconnect_attempt": {},
	"reconnecting":      {},
	"reconnect_error":   {},
	"reconnect_failed":  {},
	"removeListener":    {},
	"ping":              {},
	"pong":              {},
}

type inSocketV1 struct {
	binary, binary_     bool // the <name>_ value is the per message value...
	compress, compress_ bool // https://socket.io/blog/socket-io-1-4-0/

	ns Namespace
	id []SocketID
	in []Room
	to []Room

	tr    func() siot.Transporter
	cmprs func() io.Writer // the compress writer

	events    map[Namespace]map[Event]eventCallback
	onConnect map[Namespace]onConnectCallbackVersion1
}

func (v1 *inSocketV1) delIDs()                    { v1.id = v1.id[:0] }
func (v1 *inSocketV1) addID(id SocketID)          { v1.id = append(v1.id, id) }
func (v1 *inSocketV1) addIn(room Room)            { v1.in = append(v1.in, room) }
func (v1 *inSocketV1) addTo(room Room)            { v1.to = append(v1.to, room) }
func (v1 *inSocketV1) setNsp(namespace Namespace) { v1.ns = namespace }
func (v1 *inSocketV1) setBinary(binary bool)      { v1.binary = binary }
func (v1 *inSocketV1) setBinary_(binary bool)     { v1.binary_ = binary }
func (v1 *inSocketV1) setCompress(compress bool)  { v1.compress = compress }
func (v1 *inSocketV1) setCompress_(compress bool) { v1.compress_ = compress }

func (v1 inSocketV1) nsp() Namespace {
	if v1.ns == "" {
		v1.ns = "/"
	}
	return v1.ns
}

func (v1 *inSocketV1) clone() inSocketV1 {
	// v1.events and v1.onConnect are initialized in the NewServerV1 method
	rtn := *v1
	return rtn
}

func (v1 inSocketV1) OnConnect(callback onConnectCallbackVersion1) { v1.onConnect[v1.nsp()] = callback }

func (v1 inSocketV1) OnDisconnect(callback eventCallback) { v1.on("disconnect", callback) }

func (v1 inSocketV1) On(event Event, callback eventCallback) {
	if _, ok := v1ProtectedEventName[event]; ok {
		v1.on(event, cabk.ErrorWrap(func() error { return ErrInvalidEventName.F(event) }))
		return
	}
	v1.on(event, callback)
}

func (v1 inSocketV1) on(event Event, callback eventCallback) {
	if _, ok := v1.events[v1.nsp()]; !ok {
		v1.events[v1.nsp()] = make(map[string]eventCallback)
	}
	v1.events[v1.nsp()][event] = callback
}

// Of - sending to all clients in namespace, including sender
func (v1 inSocketV1) Of(namespace Namespace) inSocketV1 {
	rtn := v1.clone()
	rtn.setNsp(namespace)
	return rtn
}

// In - sending to all clients in room, including sender
func (v1 inSocketV1) In(room Room) inToEmit {
	rtn := v1.clone()
	rtn.addIn(room)
	return rtn
}

// To - sending to all clients in room, except sender
func (v1 inSocketV1) To(room Room) inToEmit {
	rtn := v1.clone()
	rtn.addTo(room)
	return rtn
}

// Emit - sending to all connected clients
func (v1 inSocketV1) Emit(event Event, data ...Data) error {
	transp := v1.tr().(siot.Emitter)
	ids := make(map[SocketID]struct{})

	for _, id := range transp.Sockets(v1.nsp()).IDs() {
		ids[id] = struct{}{}
	}

	v1.delIDs()
	for id := range ids {
		v1.addID(id)
	}

	return v1.emit(event, data...)
}

func (v1 inSocketV1) emit(event Event, data ...Data) error {
	// if !boolIs(v1.binary, v1.binary_) {
	// 	for _, datum := range data {
	// 		if _, ok := datum.(io.Reader); ok {
	// 			return fmt.Errorf("found binary data, only strings expected")
	// 		}
	// 	}
	// }

	callbackData, eventCallback, err := scrub(!boolIs(v1.binary, v1.binary_), event, data)
	if err != nil {
		return err
	}

	transport := v1.tr()
	if eventCallback != nil {
		v1.on(fmt.Sprintf("%s%d", ackIDEventPrefix, transport.AckID()), eventCallback)
	}

	// last := len(data) - 1

	// if callback, ok := data[last].(eventCallback); ok {
	// 	v1.on(fmt.Sprintf("%s%d", ackIDEventPrefix, transport.AckID()), callback)
	// }

	for _, id := range v1.id {
		transport.Send(id, callbackData, siop.WithType(siop.EventPacket.Byte()), siop.WithNamespace(v1.nsp()))
	}

	v1.delIDs()
	return nil
}

type SocketV1 struct {
	inSocketV1

	ID        SocketID
	Connected bool

	req *Request
}

func (v1 SocketV1) Request() *Request { return v1.req }

// In - sending to all clients in room, including sender
func (v1 SocketV1) In(room Room) inToEmit {
	rtn := v1.clone()
	rtn.addIn(room)
	return SocketV1{inSocketV1: rtn, ID: v1.ID, req: v1.req}
}

// To - sending to all clients in room, except sender
func (v1 SocketV1) To(room Room) inToEmit {
	rtn := v1.clone()
	rtn.addTo(room)
	return SocketV1{inSocketV1: rtn, ID: v1.ID, req: v1.req}
}

func (v1 SocketV1) Join(room Room) error {
	transport := v1.tr()
	return transport.Join(v1.nsp(), v1.ID, room)
}

func (v1 SocketV1) Leave(room Room) error {
	transport := v1.tr()
	return transport.Leave(v1.nsp(), v1.ID, room)
}

func (v1 SocketV1) Emit(event Event, data ...Data) error {
	if len(v1.id) > 0 {
		return v1.emit(event, data...)
	}

	transport := v1.tr().(siot.Emitter)
	ids := make(map[SocketID]struct{})

	var hasRoom bool
	for _, inRoom := range v1.in {
		// for _, id := range transport.SocketIDsFrom(v1.nsp(), inRoom) {
		rooms, err := transport.Sockets(v1.nsp()).FromRoom(inRoom)
		if err != nil {
			panic(err)
		}
		for _, id := range rooms {
			hasRoom = true
			ids[id] = struct{}{}
		}
	}

	for _, toRoom := range v1.to {
		rooms, err := transport.Sockets(v1.nsp()).FromRoom(toRoom)
		if err != nil {
			panic(err)
		}
		for _, id := range rooms {
			if id == v1.ID {
				continue // skip sending back to sender
			}
			hasRoom = true
			ids[id] = struct{}{}
		}
	}

	v1.delIDs()

	if !hasRoom {
		v1.addID(v1.ID)
	}
	for id := range ids {
		v1.addID(id)
	}

	return v1.emit(event, data...)
}

func (v1 SocketV1) Broadcast() emit {
	transport := v1.tr().(siot.Emitter)
	ids := make(map[SocketID]struct{})

	for _, id := range transport.Sockets(v1.nsp()).IDs() {
		ids[id] = struct{}{}
	}

	v1.delIDs()
	for id := range ids {
		if id == v1.ID {
			continue
		}
		v1.addID(id)
	}

	return v1
}

func (v1 SocketV1) Volatile() broadcastEmit              { return v1 } // NOT IMPLEMENTED...
func (v1 SocketV1) Compress(compress bool) broadcastEmit { v1.setCompress_(compress); return v1 }
func (v1 SocketV1) Binary(binary bool) broadcastEmit     { v1.setBinary_(binary); return v1 }
