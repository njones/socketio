package socketio

import (
	"net/http"
	"sync"

	nmem "github.com/njones/socketio/adaptor/transport/memory"
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

	v1 := v2.prev
	v1.eio = eio.NewServerV3(
		eio.WithPath(*v1.path),
		eio.WithInitialPackets(autoConnect(v1)),
	).(eio.EIOServer) // v2 uses the default engineio protocol v3
	v1.eio.With(opts...)

	v2.With(opts...)
	return v2
}

func (v2 *ServerV2) new(opts ...Option) Server {
	v2.prev = (&ServerV1{inSocketV1: inSocketV1{ʟ: new(sync.RWMutex), x: new(sync.Mutex)}}).new(opts...).(*ServerV1)
	v2.onConnect = make(map[Namespace]onConnectCallbackVersion2)

	v1 := v2.prev
	v1.run = runV2(v2)

	v1.transport = nmem.NewInMemoryTransport(siop.NewPacketV2) // v2 uses the default socketio protocol v3
	v1.setTransporter(v1.transport)

	v1.doConnectPacket = doConnectPacketV2(v2)

	v2.doBinaryEventPacket = doBinaryEventPacket(v2)
	v2.inSocketV2.prev = v1.inSocketV1.clone()

	return v2
}

func (v2 *ServerV2) With(opts ...Option) {
	v2.prev.With(opts...)
	for _, opt := range opts {
		opt(v2)
	}
}

func (v2 *ServerV2) In(room Room) inToEmit {
	rtn := v2.clone()
	rtn.setIsServer(true)
	return rtn.In(room)
}

func (v2 *ServerV2) Of(ns Namespace) inSocketV2 {
	rtn := v2.clone()
	rtn.setIsServer(true)
	return rtn.Of(ns)
}

func (v2 *ServerV2) To(room Room) inToEmit {
	rtn := v2.clone()
	rtn.setIsServer(true)
	return rtn.To(room)
}

func (v2 *ServerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) { v2.prev.ServeHTTP(w, r) }
