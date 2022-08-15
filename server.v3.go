package socketio

import (
	"net/http"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eio "github.com/njones/socketio/engineio"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

// https://socket.io/docs/v4/migrating-from-2-x-to-3-0/
// This is the revision 5 of the Socket.IO protocol, included in socket.io@3.0.0...latest.

type ServerV3 struct {
	inSocketV3

	doBinaryAckPacket func(SocketID, siot.Socket) error

	prev *ServerV2
}

func NewServerV3(opts ...Option) *ServerV3 {
	v3 := &ServerV3{}
	v3.new(opts...)
	return v3
}

func (v3 *ServerV3) new(opts ...Option) Server {
	v3.prev = (&ServerV2{}).new(opts...).(*ServerV2)
	v3.onConnect = make(map[Namespace]onConnectCallbackVersion3)

	v2 := v3.prev
	v1 := v2.prev

	v1.run = runV3(v3)
	v1.eio = eio.NewServerV4(eio.WithPath(*v1.path)).(eio.EIOServer) // v2 uses the default engineio protocol v3
	v1.transport = tmap.NewMapTransport(siop.NewPacketV4)            // v2 uses the default socketio protocol v3
	v1.protectedEventName = v3ProtectedEventName
	v1.doConnectPacket = doConnectPacketV3(v3)
	v1.doAutoReconnect = nil

	v3.doBinaryAckPacket = doBinaryAckPacket(v1)
	v3.inSocketV3.prev = v2.inSocketV2
	v3.With(opts...)
	if eioSvr, ok := v1.eio.(withOption); ok {
		eioSvr.With(v1.eio.(Server), opts...)
	}

	return v3
}

func (v3 *ServerV3) With(opts ...Option) { v1 := v3.prev.prev; v1.with(v3, opts...) }

func (v3 *ServerV3) In(room Room) inToEmit { v3.setIsServer(true); return v3.inSocketV3.In(room) }

func (v3 *ServerV3) Of(ns Namespace) inSocketV3 { v3.setIsServer(true); return v3.inSocketV3.Of(ns) }

func (v3 *ServerV3) To(room Room) inToEmit { v3.setIsServer(true); return v3.inSocketV3.To(room) }

func (v3 *ServerV3) ServeHTTP(w http.ResponseWriter, r *http.Request) { v3.prev.ServeHTTP(w, r) }
