package socketio

import siot "github.com/njones/socketio/transport"

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

func (v2 *inSocketV2) delIDs()                    { v2.prev.delIDs() }
func (v2 *inSocketV2) addID(id siot.SocketID)     { v2.prev.addID(id) }
func (v2 *inSocketV2) addIn(room Room)            { v2.prev.addIn(room) }
func (v2 *inSocketV2) addTo(room Room)            { v2.prev.addTo(room) }
func (v2 *inSocketV2) setNsp(namespace Namespace) { v2.prev.setNsp(namespace) }
func (v2 *inSocketV2) setBinary(binary bool)      { v2.prev.setBinary(binary) }
func (v2 *inSocketV2) setBinary_(binary bool)     { v2.prev.setBinary_(binary) }
func (v2 *inSocketV2) setCompress(compress bool)  { v2.prev.setCompress(compress) }
func (v2 *inSocketV2) setCompress_(compress bool) { v2.prev.setCompress_(compress) }

func (v2 inSocketV2) nsp() Namespace { return v2.prev.nsp() }

func (v2 inSocketV2) OnConnect(callback onConnectCallbackVersion2) {
	v2.onConnect[v2.nsp()] = callback
}
func (v2 inSocketV2) OnDisconnect(callback eventCallback) { v2.prev.OnDisconnect(callback) }

func (v2 inSocketV2) On(event Event, callback eventCallback) { v2.prev.On(event, callback) }

// Of - sending to all clients in namespace, including sender
func (v2 inSocketV2) Of(namespace Namespace) inSocketV2 {
	rtn := v2.clone()
	rtn.setNsp(namespace)
	return rtn
}

// In - sending to all clients in room, including sender
func (v2 inSocketV2) In(room Room) inToEmit {
	rtn := v2.clone()
	rtn.addIn(room)
	return rtn
}

// To - sending to all clients in room, except sender
func (v2 inSocketV2) To(room Room) inToEmit {
	rtn := v2.clone()
	rtn.addTo(room)
	return rtn
}

// Emit - sending to all connected clients
func (v2 inSocketV2) Emit(event Event, data ...Data) error { return v2.prev.Emit(event, data...) }

type onConnectCallbackVersion2 = func(*SocketV2) error

type SocketV2 struct {
	inSocketV2

	ID  SocketID
	req *Request
}

func (v2 *SocketV2) tr() siot.Transporter { v1 := v2.prev; return v1.tr() }

func (v2 *SocketV2) Emit(event Event, data ...Data) error { return v2.prev.Emit(event, data...) }

func (v2 *SocketV2) Broadcast() emit {
	transp := v2.tr().(siot.Emitter)
	ids := make(map[siot.SocketID]struct{})

	for _, id := range transp.Sockets(v2.nsp()).IDs() {
		ids[id] = struct{}{}
	}

	v2.delIDs()
	for id := range ids {
		v2.addID(id)
	}

	return v2
}

func (v2 *SocketV2) Volatile() emit              { return v2 } // NOT IMPLEMENTED...
func (v2 *SocketV2) Compress(compress bool) emit { v2.setCompress_(compress); return v2 }
func (v2 *SocketV2) Binary(binary bool) emit     { v2.setBinary_(binary); return v2 }
