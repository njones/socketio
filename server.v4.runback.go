package socketio

import (
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func doConnectPacketV4(v4 *ServerV4) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID SocketID, socket siot.Socket) (err error) {
		transport := v4.tr()
		transport.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		transport.(tsp).Transport(socketID).StartBuffer()
		defer func() { transport.(tsp).Transport(socketID).StopBuffer() }()

		v4.setPrefix()
		v4.setSocketID(socketID)
		v4.setNsp(socket.Namespace)

		socketV4 := &SocketV4{inSocketV4: v4.inSocketV4, req: req}
		if fn, ok := v4.onConnect[socket.Namespace]; ok {
			return fn(socketV4)
		}

		return ErrNamespaceNotFound.F(socket.Namespace)
	}
}

func runV4(v4 *ServerV4) func(r *Request, socketID SocketID) error {
	return func(r *Request, socketID SocketID) error {
		for socket := range v4.tr().Receive(socketID) {
			doV4(v4, r, socketID, socket)
		}
		return nil
	}
}

func doV4(v4 *ServerV4, r *Request, socketID SocketID, socket siot.Socket) {
	v1 := v4.prev.prev.prev

	switch socket.Type {
	case siop.ConnectPacket.Byte():
		if err := v1.doConnectPacket(r, socketID, socket); err != nil {
			v4.tr().Send(socketID, serviceError(err), siop.WithType(byte(siop.ConnectErrorPacket)))
			return
		}
		connectResponse := map[string]interface{}{"sid": socketID.String()}
		v4.tr().Send(socketID, connectResponse, siop.WithType(siop.ConnectPacket.Byte()), siop.WithNamespace(socket.Namespace))
		v4.tr().(tsp).Transport(socketID).SendBuffer()
		return
	}
	doV3(v4.prev, r, socketID, socket)
}
