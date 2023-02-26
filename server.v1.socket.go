package socketio

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	call "github.com/njones/socketio/callback"
	siop "github.com/njones/socketio/protocol"
	seri "github.com/njones/socketio/serialize"
	siot "github.com/njones/socketio/transport"
)

const serverEvent = "...*..."

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

// the embedded struct that is used to service call of the Server level values
type inSocketV1 struct {
	_socketID          SocketID // is only set if instantiated by a socket
	_socketPrefix      string
	isSender, isServer bool

	binary   bool
	compress bool // https://socket.io/blog/socket-io-1-4-0/

	tr func() siot.Transporter
	ns Namespace
	id []SocketID
	to []Room

	_uniqID map[SocketID]struct{}

	protectedEventName map[string]struct{}

	onConnect map[Namespace]onConnectCallbackVersion1
	events    map[Namespace]map[Event]map[SocketID]eventCallback

	o atomic.Value
	ʟ *sync.RWMutex
	x *sync.Mutex
}

func (v1 *inSocketV1) l() func() {
	v1.ʟ.Lock()
	return func() { v1.ʟ.Unlock() }
}

func (v1 *inSocketV1) r() func() {
	v1.ʟ.RLock()
	return func() { v1.ʟ.RUnlock() }
}

func (v1 *inSocketV1) clone() inSocketV1 {
	// v1.events and v1.onConnect are initialized in the NewServerV1 method
	defer v1.l()()

	rtn := *v1
	return rtn
}

func (v1 *inSocketV1) setTransporter(tr siot.Transporter) {
	v1.o.Store(tr)

	v1.tr = func() siot.Transporter {
		return v1.o.Load().(siot.Transporter)
	}
}
func (v1 *inSocketV1) setIsServer(isServer bool) { defer v1.l()(); v1.isServer = isServer }
func (v1 *inSocketV1) setIsSender(isSender bool) { defer v1.l()(); v1.isSender = isSender }
func (v1 *inSocketV1) setSocketID(id SocketID)   { defer v1.l()(); v1._socketID = id }
func (v1 *inSocketV1) setPrefix()                { defer v1.l()(); v1._socketPrefix = socketIDQuickPrefix() }
func (v1 *inSocketV1) setNsp(namespace Namespace) {
	defer v1.l()()

	if len(namespace) > 0 {
		if namespace[0] != '/' {
			namespace = "/" + namespace
		}
	}
	v1.ns = namespace
}
func (v1 *inSocketV1) addID(id SocketID) {
	defer v1.l()()

	if v1._uniqID == nil {
		v1._uniqID = make(map[SocketID]struct{})
	}
	if _, ok := v1._uniqID[id]; !ok {
		v1.id = append(v1.id, id)
		v1._uniqID[id] = struct{}{}
	}
}
func (v1 *inSocketV1) addTo(room Room) { defer v1.l()(); v1.to = append(v1.to, room) }

func (v1 inSocketV1) nsp() Namespace {
	defer v1.r()()

	if v1.ns == "" {
		v1.ns = "/"
	}
	return v1.ns
}
func (v1 inSocketV1) socketID() SocketID { defer v1.r()(); return v1._socketID }
func (v1 inSocketV1) prefix() string     { defer v1.r()(); return v1._socketPrefix }

func (v1 inSocketV1) OnConnect(callback onConnectCallbackVersion1) {
	v1.onConnect[v1.nsp()] = callback
}

func (v1 inSocketV1) OnDisconnect(callback func(string)) {
	v1.on("disconnect", call.FuncString(callback))
}

func (v1 inSocketV1) On(event Event, callback eventCallback) {
	if _, ok := v1.protectedEventName[event]; ok {
		v1.on(event, call.ErrorWrap(func() error { return ErrUnsupportedEventName.F(event) }))
		return
	}
	v1.on(event, callback)
}

func (v1 inSocketV1) on(event Event, callback eventCallback) {
	v1.x.Lock()
	defer v1.x.Unlock()

	if _, ok := v1.events[v1.nsp()]; !ok {
		v1.events[v1.nsp()] = make(map[string]map[SocketID]eventCallback)
	}
	if _, ok := v1.events[v1.nsp()][event]; !ok {
		v1.events[v1.nsp()][event] = make(map[SocketID]eventCallback)
	}

	socketID := v1._socketID
	if len(v1._socketID) == 0 {
		socketID = serverEvent
	}

	v1.events[v1.nsp()][event][socketID] = callback
}

// Of - sending to all clients in namespace, including sender
func (v1 inSocketV1) Of(namespace Namespace) inSocketV1 {
	rtn := v1.clone()
	rtn.setNsp(namespace)
	return rtn
}

func (v1 inSocketV1) In(room Room) inToEmit {
	return v1.To(room)
}

func (v1 inSocketV1) To(room Room) inToEmit {
	room = strings.Replace(room, v1.prefix(), socketIDPrefix, 1)
	rtn := v1.clone()
	rtn.addTo(room)
	return rtn
}

// Emit - sending to all connected clients
func (v1 inSocketV1) Emit(event Event, data ...Data) error {
	var uniqueID = map[SocketID]struct{}{}
	for _, id := range v1.id {
		uniqueID[id] = struct{}{}
	}

	transport := v1.tr().(siot.Emitter)

	if len(v1.to) == 0 && len(v1.id) == 0 {
		for _, id := range transport.Sockets(v1.nsp()).IDs() {
			if id == v1._socketID && v1.isSender {
				continue // skip sending back to sender
			}

			if _, inSet := uniqueID[id]; !inSet {
				v1.addID(id)
				uniqueID[id] = struct{}{}
			}
		}
		// send to local server ... since this is not a broadcast
		if ns, ok := v1.events[v1.nsp()]; ok {
			if events, ok := ns[event][v1._socketID]; ok {
				events.Callback(seri.Convert(data).ToInterface()...)
			}
		}
		return v1.emit(event, data...)
	}

	for _, toRoom := range v1.to {
		rooms, err := transport.Sockets(v1.nsp()).FromRoom(toRoom)
		if err != nil {
			return ErrFromRoomFailed.F(err)
		}

		for _, id := range rooms {
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

func (v1 inSocketV1) emit(event Event, data ...Data) error {
	hasBin, callbackData, eventCallback, err := scrub(v1.binary, event, data)
	if err != nil {
		return err
	}

	transport := v1.tr()
	for _, id := range v1.id {
		opts := []siop.Option{siop.WithNamespace(v1.nsp())}
		if hasBin {
			if eventCallback != nil {
				opts = append(opts, siop.WithType(siop.BinaryAckPacket.Byte()))
			} else {
				opts = append(opts, siop.WithType(siop.BinaryEventPacket.Byte()))
			}
		} else {
			opts = append(opts, siop.WithType(siop.EventPacket.Byte()))
		}
		if eventCallback != nil {
			ackID := transport.AckID()
			v1.on(fmt.Sprintf("%s%d", ackIDEventPrefix, ackID), eventCallback)
			opts = append(opts, siop.WithAckID(ackID))
		}
		transport.Send(id, callbackData, opts...)
	}

	return nil
}

// SocketV1 is the returned socket
type SocketV1 struct {
	inSocketV1

	Connected bool

	req *Request
}

func (v1 *SocketV1) ID() SocketID      { return SocketID(v1.prefix()) + v1.socketID() }
func (v1 *SocketV1) Request() *Request { return v1.req }

func (v1 *SocketV1) Emit(event Event, data ...Data) error {
	v1.addID(v1.socketID())
	return v1.emit(event, data...)
}

func (v1 *SocketV1) Join(room Room) error {
	transport := v1.tr()
	return transport.Join(v1.nsp(), v1.socketID(), room)
}

func (v1 *SocketV1) Leave(room Room) error {
	transport := v1.tr()
	return transport.Leave(v1.nsp(), v1.socketID(), room)
}

func (v1 *SocketV1) Broadcast() emit                      { v1.setIsSender(true); return v1.inSocketV1 }
func (v1 *SocketV1) Volatile() broadcastEmit              { return v1 } // NOT IMPLEMENTED...
func (v1 *SocketV1) Compress(compress bool) broadcastEmit { return v1 } // NOT IMPLEMENTED...
