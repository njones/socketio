package socketio

import (
	"net/http"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eio "github.com/njones/socketio/engineio"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

// The 4th revision (included in socket.io@1.0.3...2.x.x) can be found here: https://github.com/socketio/socket.io-protocol/tree/v4
type ServerV2 struct {
	inSocketV2

	doBinaryEventPacket func(SocketID, siot.Socket) error

	prev *ServerV1
}

func NewServerV2(opts ...Option) *ServerV2 {
	v2 := &ServerV2{}
	v2.new(opts...)
	return v2
}

func (v2 *ServerV2) new(opts ...Option) Server {
	v2.prev = (&ServerV1{}).new(opts...).(*ServerV1)
	v2.onConnect = make(map[Namespace]onConnectCallbackVersion2)

	v1 := v2.prev

	v1.run = runV2(v2)
	v1.eio = eio.NewServerV3(eio.WithPath(*v1.path)).(eio.EIOServer) // v2 uses the default engineio protocol v3
	v1.transport = tmap.NewMapTransport(siop.NewPacketV2)            // v2 uses the default socketio protocol v3
	v1.doConnectPacket = doConnectPacketV2(v2)

	v2.doBinaryEventPacket = doBinaryEventPacket(v2)
	v2.inSocketV2.prev = v1.inSocketV1

	v2.With(opts...)
	if eioSvr, ok := v1.eio.(withOption); ok {
		eioSvr.With(v1.eio.(Server), opts...)
	}

	return v2
}

func (v2 *ServerV2) With(opts ...Option) { v1 := v2.prev; v1.with(v2, opts...) }

func (v2 *ServerV2) In(room Room) inToEmit { v2.setIsServer(true); return v2.inSocketV2.In(room) }

func (v2 *ServerV2) Of(ns Namespace) inSocketV2 { v2.setIsServer(true); return v2.inSocketV2.Of(ns) }

func (v2 *ServerV2) To(room Room) inToEmit { v2.setIsServer(true); return v2.inSocketV2.To(room) }

func (v2 *ServerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) { v2.prev.ServeHTTP(w, r) }
