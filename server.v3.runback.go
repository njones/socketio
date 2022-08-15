package socketio

import (
	"fmt"

	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func doConnectPacketV3(v3 *ServerV3) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID SocketID, socket siot.Socket) (err error) {
		transport := v3.tr()
		transport.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		// transport.(tsp).Transport(socketID).StartBuffer()
		// defer func() { transport.(tsp).Transport(socketID).StopBuffer() }()

		v3.setPrefix()
		v3.setSocketID(socketID)
		v3.setNsp(socket.Namespace)

		socketV3 := &SocketV3{inSocketV3: v3.inSocketV3, req: req}
		if fn, ok := v3.onConnect[socket.Namespace]; ok {
			return fn(socketV3)
		}

		return ErrNamespaceNotFound.F(socket.Namespace)
	}
}

func doBinaryAckPacket(v1 *ServerV1) func(SocketID, siot.Socket) error {
	return func(socketID SocketID, socket siot.Socket) (err error) {
		event := fmt.Sprintf("%s%d", ackIDEventPrefix, socket.AckID)

		switch data := socket.Data.(type) {
		case []interface{}:
			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				return fn.Callback(data...)
			}
			if fn, ok := v1.events[socket.Namespace][event][serverEvent]; ok {
				return fn.Callback(data...)
			}
		case []string:
			event := data[0]
			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				err = fn.Callback(stoi(data[1:])...)
			}
		default:
			return ErrInvalidPacketTypeExpected.F(socket.Data)
		}
		return err
	}
}

func runV3(v3 *ServerV3) func(r *Request, socketID SocketID) error {
	return func(r *Request, socketID SocketID) error {
		for socket := range v3.tr().Receive(socketID) {
			doV3(v3, r, socketID, socket)
		}
		return nil
	}
}

func doV3(v3 *ServerV3, r *Request, socketID SocketID, socket siot.Socket) {
	switch socket.Type {
	case siop.BinaryAckPacket.Byte():
		if err := v3.doBinaryAckPacket(socketID, socket); err != nil {
			v3.tr().Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
		}
	}
	doV2(v3.prev, r, socketID, socket)
}
