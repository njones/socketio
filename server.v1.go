package socketio

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	nmem "github.com/njones/socketio/adaptor/transport/memory"
	eio "github.com/njones/socketio/engineio"
	eiot "github.com/njones/socketio/engineio/transport"
	erro "github.com/njones/socketio/internal/errors"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

// The 3rd revision (included in socket.io@1.0.0...1.0.2) can be found here: https://github.com/socketio/socket.io-protocol/tree/v3

// https://socket.io/blog/introducing-socket-io-1-0
// https://socket.io/blog/socket-io-1-4-0/
// https://socket.io/blog/socket-io-1-4-5/

// ServerV1 is the same as the javascript SocketIO v1.0 server.
type ServerV1 struct {
	inSocketV1

	run                func(socketID SocketID, req *Request) error
	doConnectPacket    func(socketID SocketID, socket siot.Socket, req *Request) error
	doDisconnectPacket func(socketID SocketID, socket siot.Socket, req *Request) error
	doEventPacket      func(socketID SocketID, socket siot.Socket) error
	doAckPacket        func(socketID SocketID, socket siot.Socket) error

	path *string

	ctx context.Context
	eio eio.EIOServer

	transport siot.Transporter
}

// NewServerV1 returns a new v1.0 SocketIO server
func NewServerV1(opts ...Option) *ServerV1 {
	v1 := &ServerV1{inSocketV1: inSocketV1{ʟ: new(sync.RWMutex), x: new(sync.Mutex)}}
	v1.new(opts...)

	v1.eio = eio.NewServerV2(
		eio.WithPath(*v1.path),
		eio.WithInitialPackets(autoConnect(v1)),
	).(eio.EIOServer)
	v1.eio.With(opts...)

	v1.With(opts...)
	return v1
}

// new returns a new ServerV1 with the different options. This should be called
// when setting up a new server, as it sets up the defaults. The defaults can
// be over written by the Options. Note that the Options can also include options
// that can be applied to the underlining engineIO server.
func (v1 *ServerV1) new(opts ...Option) Server {
	v1.run = runV1(v1)
	v1.doConnectPacket = doConnectPacket(v1)
	v1.doDisconnectPacket = doDisconnectPacket(v1)
	v1.doEventPacket = doEventPacket(v1)
	v1.doAckPacket = doAckPacket(v1)

	v1.ns = "/"
	v1.path = ampersand("/socket.io/")
	v1.events = make(map[Namespace]map[Event]map[SocketID]eventCallback)
	v1.onConnect = make(map[Namespace]onConnectCallbackVersion1)

	v1.protectedEventName = v1ProtectedEventName

	v1.transport = nmem.NewInMemoryTransport(siop.NewPacketV2) // set the default transport

	v1.inSocketV1.binary = true   // for the v1 implementation this always is set to true
	v1.inSocketV1.compress = true // for the v1 implementation this always is set to true

	v1.inSocketV1.setTransporter(v1.transport)

	return v1
}

func (v1 *ServerV1) With(opts ...Option) {
	for _, opt := range opts {
		opt(v1)
	}
}

func (v1 *ServerV1) In(room Room) inToEmit {
	rtn := v1.clone()
	rtn.setIsServer(true)
	return rtn.In(room)
}

func (v1 *ServerV1) Of(ns Namespace) inSocketV1 {
	rtn := v1.clone()
	v1.setIsServer(true)
	return rtn.Of(ns)
}

func (v1 *ServerV1) To(room Room) inToEmit {
	rtn := v1.clone()
	rtn.setIsServer(true)
	return rtn.To(room)
}

// ServeHTTP is the interface for applying a http request/response cycle. This handles
// errors that can be provided by the underlining serveHTTP method that uses errors.
func (v1 *ServerV1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if v1.path != nil && !strings.HasPrefix(r.URL.Path, *v1.path) { // lock to the default socketio path if present
		return
	}

	ctx := r.Context()
	if v1.ctx != nil {
		ctx = v1.ctx
	}

	var eState erro.State
	if err := v1.serveHTTP(w, r.WithContext(ctx)); err != nil {
		switch {
		case errors.As(err, &eState):
			return
		case errors.Is(err, erro.HTTPStatusError400):
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// serveHTTP is the same as ServeHTTP but uses errors to break out of request cycles that
// have an error. The response is handled in the upper ServeHTTP method.
func (v1 *ServerV1) serveHTTP(w http.ResponseWriter, r *http.Request) (err error) {
	eioTransport, err := v1.eio.ServeTransport(w, r)
	if err != nil {
		return err
	}

	go func(e eiot.Transporter) {
		for {
			sessionId, ok := <-e.ReceiveTimeout()
			if ok {
				socketId := v1.transport.GetSocketID(sessionId)
				if socketId != nil && !v1.tr().IsDisconnected(*socketId) {
					for namespaceId, namespaces := range v1.events {
						if events, ok := namespaces[OnDisconnectEvent]; ok {
							if fn, ok := events[*socketId]; ok {
								v1.tr().Leave(namespaceId, *socketId, socketIDPrefix+socketId.String())
								fn.Callback("client namespace disconnect.")
							}
						}
					}
				}
				return
			}
		}
	}(eioTransport)

	sid, err := v1.transport.Add(eioTransport)
	if err != nil {
		return err
	}
	v1.setSocketID(sid)

	return v1.run(sid, sioRequest(r))
}
