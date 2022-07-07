package socketio

import (
	"fmt"
	"io"

	cabk "github.com/njones/socketio/callback"
	siop "github.com/njones/socketio/protocol"
	"github.com/njones/socketio/session"
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

type inInSocketV1 struct{ inSocketV1 }

func (v1 inInSocketV1) In(room Room) inToEmit { return v1.To(room) }

// With takes in a server version and applies Options to that server object.
func (v1 *ServerV1) In(room Room) inToEmit { return v1.To(room) }

// the embeded struct that is used to service call of the Server level values
type inSocketV1 struct {
	binary              bool
	compress, compress_ bool // https://socket.io/blog/socket-io-1-4-0/

	keepIdx int

	ns Namespace
	id []SocketID
	to []Room

	tr    func() siot.Transporter
	cmprs func() io.Writer // the compress writer

	events    map[Namespace]map[Event]eventCallback
	onConnect map[Namespace]onConnectCallbackVersion1
}

func (v1 *inSocketV1) delIDs()                    { v1.id = v1.id[:0] }
func (v1 *inSocketV1) addID(id SocketID)          { v1.id = append(v1.id, id) }
func (v1 *inSocketV1) addTo(room Room)            { v1.to = append(v1.to, room) }
func (v1 *inSocketV1) setNsp(namespace Namespace) { v1.ns = namespace }
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
func (v1 inSocketV1) To(room Room) inToEmit {

	// Check to see if we're going to send to a socket
	transport := v1.tr().(siot.Emitter)
	for _, id := range transport.Sockets(v1.nsp()).IDs() {
		if session.ID(room) == id {
			rtn := v1.clone()
			rtn.addID(id)
			rtn.keepIdx = len(rtn.id) // this will allow skipping in the Emit method
			return inInSocketV1{rtn}
		}
	}

	rtn := v1.clone()
	rtn.addTo(room)
	return inInSocketV1{rtn}
}

// Emit - sending to all connected clients
func (v1 inSocketV1) Emit(event Event, data ...Data) error {
	if len(v1.id) < v1.keepIdx {
		return ErrBadSendToSocketIndex
	}

	if len(v1.id[v1.keepIdx:]) > 0 {
		return v1.emit(event, data...)
	}

	transport := v1.tr().(siot.Emitter)

	if len(v1.id) == 0 && len(v1.to) == 0 {
		for _, id := range transport.Sockets(v1.nsp()).IDs() {
			v1.addID(id)
		}
		return v1.emit(event, data...)
	}

	var dedupID = map[session.ID]struct{}{}
	for _, toRoom := range v1.to {
		rooms, err := transport.Sockets(v1.nsp()).FromRoom(toRoom)
		if err != nil {
			panic(err)
		}

		for _, id := range rooms {
			if _, inSet := dedupID[id]; !inSet {
				v1.addID(id)
				dedupID[id] = struct{}{}
			}
		}
	}

	return v1.emit(event, data...)
}

func (v1 inSocketV1) emit(event Event, data ...Data) error {
	callbackData, eventCallback, err := scrub(v1.binary, event, data)
	if err != nil {
		return err
	}

	transport := v1.tr()
	if eventCallback != nil {
		v1.on(fmt.Sprintf("%s%d", ackIDEventPrefix, transport.AckID()), eventCallback)
	}

	for _, id := range v1.id {
		transport.Send(id, callbackData, siop.WithType(siop.EventPacket.Byte()), siop.WithNamespace(v1.nsp()))
	}

	return nil
}

// SocketV1 is the returned socket
type SocketV1 struct {
	inSocketV1

	ID        SocketID
	Connected bool

	req *Request
}

func (v1 SocketV1) Request() *Request { return v1.req }

// To - sending to all clients in room, except sender
func (v1 SocketV1) To(room Room) toEmit {
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
	if len(v1.id) < v1.keepIdx {
		return ErrBadSendToSocketIndex
	}

	if len(v1.id[v1.keepIdx:]) > 0 {
		return v1.emit(event, data...)
	}

	transport := v1.tr().(siot.Emitter)

	// if we are not sending to clients in a room
	// then we're just sending back to the current client...
	if len(v1.id) == 0 && len(v1.to) == 0 {
		v1.addID(v1.ID)
		return v1.emit(event, data...)
	}

	var dedup = map[session.ID]struct{}{}
	for _, toRoom := range v1.to {
		rooms, err := transport.Sockets(v1.nsp()).FromRoom(toRoom)
		if err != nil {
			panic(err)
		}
		for _, id := range rooms {
			if id == v1.ID {
				continue // skip sending back to sender
			}
			if _, isSet := dedup[id]; !isSet {
				v1.addID(id)
				dedup[id] = struct{}{}
			}
		}
	}

	return v1.emit(event, data...)
}

func (v1 SocketV1) Broadcast() emit {
	// is the [broadcast].Emit function...

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
