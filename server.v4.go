package socketio

import (
	"net/http"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eio "github.com/njones/socketio/engineio"
	siop "github.com/njones/socketio/protocol"
)

// https://socket.io/docs/v4/migrating-from-3-x-to-4-0/

type ServerV4 struct {
	inSocketV4

	prev *ServerV3
}

func NewServerV4(opts ...Option) *ServerV4 {
	v4 := &ServerV4{}
	v4.new(opts...)
	return v4
}

func (v4 *ServerV4) new(opts ...Option) Server {
	v4.prev = (&ServerV3{}).new(opts...).(*ServerV3)
	v4.onConnect = make(map[Namespace]onConnectCallbackVersion4)

	v3 := v4.prev
	v2 := v3.prev
	v1 := v2.prev

	v1.run = runV4(v4)
	v1.eio = eio.NewServerV5(eio.WithPath(*v1.path)).(eio.EIOServer)
	v1.transport = tmap.NewMapTransport(siop.NewPacketV5)
	v1.protectedEventName = v4ProtectedEventName
	v1.doConnectPacket = doConnectPacketV4(v4)

	v4.inSocketV4.prev = v3.inSocketV3
	v4.With(opts...)
	if eioSvr, ok := v1.eio.(withOption); ok {
		eioSvr.With(v1.eio.(Server), opts...)
	}

	return v4
}

func (v4 *ServerV4) With(opts ...Option) { v1 := v4.prev.prev.prev; v1.with(v4, opts...) }

func (v4 *ServerV4) Except(room ...Room) innTooExceptEmit {
	v4.setIsServer(true)
	return v4.inSocketV4.Except(room...)
}

func (v4 *ServerV4) In(room ...Room) innTooExceptEmit {
	v4.setIsServer(true)
	return v4.inSocketV4.In(room...)
}

func (v4 *ServerV4) Of(namespace Namespace) inSocketV4 {
	v4.setIsServer(true)
	return v4.inSocketV4.Of(namespace)
}

func (v4 *ServerV4) To(room ...Room) innTooExceptEmit {
	v4.setIsServer(true)
	return v4.inSocketV4.To(room...)
}

func (v4 *ServerV4) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v1 := v4.prev.prev.prev
	v1.ServeHTTP(w, r)
}
