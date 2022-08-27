package socketio

import (
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func doConnectPacketV2(v2 *ServerV2) func(SocketID, siot.Socket, *Request) error {
	return func(socketID SocketID, socket siot.Socket, req *Request) (err error) {
		v2.tr().Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		v2.setPrefix()
		v2.setSocketID(socketID)
		v2.setNsp(socket.Namespace)

		if fn, ok := v2.onConnect[socket.Namespace]; ok {
			return fn(&SocketV2{inSocketV2: v2.inSocketV2, req: req})
		}
		return ErrBadOnConnectSocket
	}
}

func doBinaryEventPacket(v2 *ServerV2) func(SocketID, siot.Socket) error {
	v1 := v2.prev
	return func(socketID SocketID, socket siot.Socket) (err error) {

		switch data := socket.Data.(type) {
		case []interface{}:
			event, ok := data[0].(string)
			if !ok {
				return ErrOnBinaryEvent.F(data)
			}
			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				return fn.Callback(data[1:]...)
			}
			if fn, ok := v1.events[socket.Namespace][event][serverEvent]; ok {
				return fn.Callback(data[1:]...)
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

func runV2(v2 *ServerV2) func(SocketID, *Request) error {
	return func(socketID SocketID, req *Request) error {
		for socket := range v2.tr().Receive(socketID) {
			doV2(v2, socketID, socket, req)
		}
		return nil
	}
}

func doV2(v2 *ServerV2, socketID SocketID, socket siot.Socket, req *Request) {
	switch socket.Type {
	case siop.BinaryEventPacket.Byte():
		if err := v2.doBinaryEventPacket(socketID, socket); err != nil {
			v2.tr().Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
		}
		return
	}
	doV1(v2.prev, socketID, socket, req)
}
