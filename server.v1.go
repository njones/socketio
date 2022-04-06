package socketio

import (
	"context"
	"errors"
	"net/http"
	"strings"

	tmap "github.com/njones/socketio/adaptor/transport/map"
	eio "github.com/njones/socketio/engineio"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

type ServerV1 struct {
	inSocketV1

	run                func(req *Request, socketID SocketID) error
	doConnectPacket    func(req *Request, socketID SocketID, socket siot.Socket) error
	doDisconnectPacket func(req *Request, socketID SocketID, socket siot.Socket) error
	doEventPacket      func(socket siot.Socket) error
	doAckPacket        func(socket siot.Socket) error

	ctx context.Context

	path *string

	eio eio.EIOServer

	transport siot.Transporter
}

func NewServerV1(opts ...Option) *ServerV1 {
	v1 := &ServerV1{}
	v1.new(opts...)
	return v1
}

func (v1 *ServerV1) new(opts ...Option) Server {
	v1.run = runV1(v1)
	v1.doConnectPacket = doConnectPacket(v1)
	v1.doDisconnectPacket = doDisconnectPacket(v1)
	v1.doEventPacket = doEventPacket(v1)
	v1.doAckPacket = doAckPacket(v1)

	v1.ns = "/"
	v1.path = amp("/socket.io/")
	v1.events = make(map[Namespace]map[Event]eventCallback)
	v1.onConnect = make(map[Namespace]onConnectCallbackVersion1)

	v1.eio = eio.NewServerV2(eio.WithPath(*v1.path)).(eio.EIOServer)
	v1.transport = tmap.NewMapTransport(siop.NewPacketV2) // set the default transport

	v1.inSocketV1.binary = true   // for the v1 implentation this always is set to true
	v1.inSocketV1.compress = true // for the v1 implentation this always is set to true

	v1.With(v1, opts...)
	if eioSvr, ok := v1.eio.(withOption); ok {
		eioSvr.With(v1.eio.(Server), opts...)
	}

	v1.inSocketV1.tr = func() siot.Transporter { return v1.transport }

	return v1
}

func (v2 *ServerV1) With(svr Server, opts ...Option) {
	for _, opt := range opts {
		opt(svr)
	}
}

func (v1 *ServerV1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if v1.path != nil && !strings.HasPrefix(r.URL.Path, *v1.path) { // lock to the default socketio path if present
		return
	}

	ctx := r.Context()
	if v1.ctx != nil {
		ctx = v1.ctx
	}

	if err := v1.serveHTTP(w, r.WithContext(ctx)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (v1 *ServerV1) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	eioTransport, err := v1.eio.ServeTransport(w, r)
	if err != nil {
		if errors.Is(err, eio.EOH) {
			_, err = v1.transport.Add(eioTransport)
		}
		return err
	}

	socketID, err := v1.transport.Add(eioTransport)
	if err != nil {
		return err
	}

	return v1.run(sioRequest(r), socketID)
}
