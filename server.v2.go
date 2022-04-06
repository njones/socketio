package socketio

import (
	"net/http"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eio "github.com/njones/socketio/engineio"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

type ServerV2 struct {
	inSocketV2

	doBinaryEventPacket func(socket siot.Socket) error

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
	v2.doBinaryEventPacket = doBinaryEventPacket(v2)

	v1 := v2.prev
	v1.doConnectPacket = doConnectPacketV2(v2)
	v1.run = runV2(v2)

	v1.eio = eio.NewServerV3(eio.WithPath(*v1.path)).(eio.EIOServer) // v2 uses the default engineio protocol v3
	v1.transport = tmap.NewMapTransport(siop.NewPacketV2)            // v2 uses the default socketio protocol v3

	v2.inSocketV2.prev = v1.inSocketV1
	v1.With(v2, opts...)

	return v2
}

func (v2 *ServerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v1 := v2.prev
	v1.ServeHTTP(w, r)
}
