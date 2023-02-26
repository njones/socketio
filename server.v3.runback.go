package socketio

import (
	"fmt"

	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func doConnectPacketV3(v3 *ServerV3) func(SocketID, siot.Socket, *Request) error {
	return func(socketID SocketID, socket siot.Socket, req *Request) (err error) {
		unlock := v3.prev.prev.r()
		tr := v3.tr()
		unlock()

		tr.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		v3.setPrefix()
		v3.setSocketID(socketID)
		v3.setNsp(socket.Namespace)

		if fn, ok := v3.onConnect[socket.Namespace]; ok {
			return fn(&SocketV3{inSocketV3: v3.inSocketV3.clone(), req: req})
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
			return ErrUnexpectedBinaryData.F(socket.Data)
		}
		return err
	}
}

func runV3(v3 *ServerV3) func(SocketID, *Request) error {
	return func(socketID SocketID, req *Request) error {
		unlock := v3.prev.prev.r()
		tr := v3.tr()
		unlock()

		for socket := range tr.Receive(socketID) {
			if err := doV3(v3, socketID, socket, req); err != nil {
				return err
			}
		}
		return nil
	}
}

func doV3(v3 *ServerV3, socketID SocketID, socket siot.Socket, req *Request) error {
	switch socket.Type {
	case siop.BinaryAckPacket.Byte():
		if err := v3.doBinaryAckPacket(socketID, socket); err != nil {
			v3.tr().Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
		}
		return nil
	}
	return doV2(v3.prev, socketID, socket, req)
}
