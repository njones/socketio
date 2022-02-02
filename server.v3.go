package socketio

import (
	"net/http"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eio "github.com/njones/socketio/engineio"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

type ServerV3 struct {
	inSocketV3

	doBinaryEventPacket func(socket siot.Socket) error

	prev *ServerV2
}

func NewServerV3(opts ...Option) *ServerV3 {
	v3 := &ServerV3{}
	v3.new(opts...)
	return v3
}

func (v3 *ServerV3) new(opts ...Option) Server {
	v3.prev = (&ServerV2{}).new(opts...).(*ServerV2)
	v3.onConnect = make(map[Namespace]EventCallbackV3)
	// v3.doBinaryEventPacket = doBinaryEventPacket(v3)
	// v3.doConnectPacket = doConnectPacketV2(v3)

	v1 := v3.prev.prev
	v1.run = runV3(v3)

	v1.eio = eio.NewServerV3(eio.WithPath(*v1.path)).(eio.EIOServer) // v2 uses the default engineio protocol v3
	v1.transport = tmap.NewMapTransport(siop.NewPacketV4)            // v2 uses the default socketio protocol v3

	v1.With(v3, opts...)

	return v3
}

func (v3 *ServerV3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v1 := v3.prev.prev
	v1.ServeHTTP(w, r)
}
