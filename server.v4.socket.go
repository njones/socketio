package socketio

import siot "github.com/njones/socketio/transport"

type inSocketV4 struct {
	onConnect map[Namespace]OnConnectCbV4

	prev inSocketV3
}

func (v4 *inSocketV4) clone() inSocketV4 {
	rtn := *v4
	rtn.prev.prev.prev = v4.prev.prev.prev.clone()
	// rtn.onConnect is a map that gets copied by reference
	return rtn
}

func (v4 *inSocketV4) delIDs()                    { v4.prev.delIDs() }
func (v4 *inSocketV4) addID(id siot.SocketID)     { v4.prev.addID(id) }
func (v4 *inSocketV4) addIn(room Room)            { v4.prev.addIn(room) }
func (v4 *inSocketV4) addTo(room Room)            { v4.prev.addTo(room) }
func (v4 *inSocketV4) setNsp(namespace Namespace) { v4.prev.setNsp(namespace) }
func (v4 *inSocketV4) setBinary(binary bool)      { v4.prev.setBinary(binary) }
func (v4 *inSocketV4) setBinary_(binary bool)     { v4.prev.setBinary_(binary) }
func (v4 *inSocketV4) setCompress(compress bool)  { v4.prev.setCompress(compress) }
func (v4 *inSocketV4) setCompress_(compress bool) { v4.prev.setCompress_(compress) }

func (v4 inSocketV4) nsp() Namespace { return v4.prev.nsp() }

func (v4 inSocketV4) OnConnect(callback OnConnectCbV4) {
	v4.onConnect[v4.nsp()] = callback
}
func (v4 inSocketV4) OnDisconnect(callback EventCb) { v4.prev.OnDisconnect(callback) }

func (v4 inSocketV4) On(event Event, callback EventCb) { v4.prev.On(event, callback) }

// Of - sending to all clients in namespace, including sender
func (v4 inSocketV4) Of(namespace Namespace) inSocketV4 {
	rtn := v4.clone()
	rtn.setNsp(namespace)
	return rtn
}

// In - sending to all clients in room, including sender
func (v4 inSocketV4) In(room Room) InToEmit {
	rtn := v4.clone()
	rtn.addIn(room)
	return rtn
}

// To - sending to all clients in room, except sender
func (v4 inSocketV4) To(room Room) InToEmit {
	rtn := v4.clone()
	rtn.addTo(room)
	return rtn
}

// Emit - sending to all connected clients
func (v4 inSocketV4) Emit(event Event, data ...Data) error { return v4.prev.Emit(event, data...) }

type OnConnectCbV4 func(*SocketV4) error

type SocketV4 struct {
	inSocketV4

	ID  SocketID
	req *Request
}

func (v4 *SocketV4) tr() siot.Transporter { v1 := v4.prev.prev.prev; return v1.tr() }

func (v4 *SocketV4) Emit(event Event, data ...Data) error { return v4.prev.Emit(event, data...) }

func (v4 *SocketV4) Broadcast() Emit {
	transp := v4.tr().(siot.Emitter)
	ids := make(map[siot.SocketID]struct{})

	for _, id := range transp.Sockets(v4.nsp()).IDs() {
		ids[id] = struct{}{}
	}

	v4.delIDs()
	for id := range ids {
		v4.addID(id)
	}

	return v4
}

func (v4 *SocketV4) Volatile() Emit              { return v4 } // NOT IMPLEMENTED...
func (v4 *SocketV4) Compress(compress bool) Emit { v4.setCompress_(compress); return v4 }
func (v4 *SocketV4) Binary(binary bool) Emit     { v4.setBinary_(binary); return v4 }
