package socketio

import (
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func doConnectPacketV2(v2 *ServerV2) func(SocketID, siot.Socket, *Request) error {
	return func(socketID SocketID, socket siot.Socket, req *Request) (err error) {
		unlock := v2.prev.r()
		tr := v2.tr()
		unlock()

		tr.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		v2.setPrefix()
		v2.setSocketID(socketID)
		v2.setNsp(socket.Namespace)

		if fn, ok := v2.onConnect[socket.Namespace]; ok {
			return fn(&SocketV2{inSocketV2: v2.inSocketV2.clone(), req: req})
		}
		return ErrOnConnectSocket
	}
}

func doBinaryEventPacket(v2 *ServerV2) func(SocketID, siot.Socket) error {
	v1 := v2.prev
	return func(socketID SocketID, socket siot.Socket) (err error) {
		type callbackAck interface {
			CallbackAck(...interface{}) []interface{}
		}

		switch data := socket.Data.(type) {
		case []interface{}:
			event, ok := data[0].(string)
			if !ok {
				return ErrUnknownBinaryEventName.F(data)
			}
			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				if socket.AckID > 0 {
					if fn, ok := fn.(callbackAck); ok {
						vals := fn.CallbackAck(data[1:]...)
						return v1.tr().Send(socketID, vals, siop.WithNamespace(socket.Namespace), siop.WithAckID(socket.AckID), siop.WithType(byte(siop.BinaryAckPacket)))
					}
				}
				return fn.Callback(data[1:]...)
			}
			if fn, ok := v1.events[socket.Namespace][event][serverEvent]; ok {
				if socket.AckID > 0 {
					if fn, ok := fn.(callbackAck); ok {
						vals := fn.CallbackAck(data[1:]...)
						return v1.tr().Send(socketID, vals, siop.WithNamespace(socket.Namespace), siop.WithAckID(socket.AckID), siop.WithType(byte(siop.BinaryAckPacket)))
					}
				}
				return fn.Callback(data[1:]...)
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

func runV2(v2 *ServerV2) func(SocketID, *Request) error {
	return func(socketID SocketID, req *Request) error {
		unlock := v2.prev.r()
		tr := v2.tr()
		unlock()

		for socket := range tr.Receive(socketID) {
			if err := doV2(v2, socketID, socket, req); err != nil {
				return err
			}
		}
		return nil
	}
}

func doV2(v2 *ServerV2, socketID SocketID, socket siot.Socket, req *Request) error {
	switch socket.Type {
	case siop.BinaryEventPacket.Byte():
		if err := v2.doBinaryEventPacket(socketID, socket); err != nil {
			v2.tr().Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
		}
		return nil
	}
	return doV1(v2.prev, socketID, socket, req)
}
