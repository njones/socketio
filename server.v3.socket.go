package socketio

import siot "github.com/njones/socketio/transport"

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

func (v3 *inSocketV3) delIDs()                    { v3.prev.delIDs() }
func (v3 *inSocketV3) addID(id siot.SocketID)     { v3.prev.addID(id) }
func (v3 *inSocketV3) addTo(room Room)            { v3.prev.addTo(room) }
func (v3 *inSocketV3) setNsp(namespace Namespace) { v3.prev.setNsp(namespace) }
func (v3 *inSocketV3) setBinary(binary bool)      { v3.prev.setBinary(binary) }
func (v3 *inSocketV3) setBinary_(binary bool)     { v3.prev.setBinary_(binary) }
func (v3 *inSocketV3) setCompress(compress bool)  { v3.prev.setCompress(compress) }
func (v3 *inSocketV3) setCompress_(compress bool) { v3.prev.setCompress_(compress) }

func (v3 inSocketV3) nsp() Namespace { return v3.prev.nsp() }

func (v3 inSocketV3) OnConnect(callback onConnectCallbackVersion3) {
	v3.onConnect[v3.nsp()] = callback
}
func (v3 inSocketV3) OnDisconnect(callback eventCallback) { v3.prev.OnDisconnect(callback) }

func (v3 inSocketV3) On(event Event, callback eventCallback) { v3.prev.On(event, callback) }

// Of - sending to all clients in namespace, including sender
func (v3 inSocketV3) Of(namespace Namespace) inSocketV3 {
	rtn := v3.clone()
	rtn.setNsp(namespace)
	return rtn
}

// In - sending to all clients in room, including sender
func (v3 inSocketV3) In(room Room) inToEmit {
	rtn := v3.clone()
	rtn.addTo(room) // addIn
	return rtn
}

// To - sending to all clients in room, except sender
func (v3 inSocketV3) To(room Room) inToEmit {
	rtn := v3.clone()
	rtn.addTo(room)
	return rtn
}

// Emit - sending to all connected clients
func (v3 inSocketV3) Emit(event Event, data ...Data) error { return v3.prev.Emit(event, data...) }

type onConnectCallbackVersion3 = func(*SocketV3) error

type SocketV3 struct {
	inSocketV3

	ID  SocketID
	req *Request
}

func (v3 *SocketV3) tr() siot.Transporter { v1 := v3.prev.prev; return v1.tr() }

func (v3 *SocketV3) Emit(event Event, data ...Data) error { return v3.prev.Emit(event, data...) }

func (v3 *SocketV3) Broadcast() emit {
	transp := v3.tr().(siot.Emitter)
	ids := make(map[siot.SocketID]struct{})

	for _, id := range transp.Sockets(v3.nsp()).IDs() {
		ids[id] = struct{}{}
	}

	v3.delIDs()
	for id := range ids {
		v3.addID(id)
	}

	return v3
}

func (v3 *SocketV3) Volatile() emit              { return v3 } // NOT IMPLEMENTED...
func (v3 *SocketV3) Compress(compress bool) emit { v3.setCompress_(compress); return v3 }
func (v3 *SocketV3) Binary(binary bool) emit     { v3.setBinary_(binary); return v3 }
