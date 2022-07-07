package socketio

import (
	"fmt"

	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

// runV1 are the callbacks that are used for version 1 of the server based on the
// receieve of the transport and the packet type. This can be different for the
// different server versions.
func runV1(v1 *ServerV1) func(*Request, siot.SocketID) error {
	return func(r *Request, socketID siot.SocketID) error {

		for socket := range v1.transport.Receive(socketID) {
			switch socket.Type {
			case siop.ConnectPacket.Byte():
				if err := v1.doConnectPacket(r, socketID, socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
					return err
				}
			case siop.DisconnectPacket.Byte():
				if err := v1.doDisconnectPacket(r, socketID, socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
					return err
				}
				return nil
			case siop.EventPacket.Byte():
				if err := v1.doEventPacket(socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
					return err
				}
			case siop.AckPacket.Byte():
				if err := v1.doAckPacket(socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
					return err
				}
			case siop.ErrorPacket.Byte():
				if e, ok := socket.Data.(error); ok {
					return e
				}
			default:
				err := ErrInvalidPacketType.F("v1", socket)
				v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
				return err
			}
		}

		return nil // should never reach here
	}
}

// doConnectPacket the function
func doConnectPacket(v1 *ServerV1) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID SocketID, socket siot.Socket) (err error) {
		v1.transport.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		socketV1 := &SocketV1{inSocketV1: v1.inSocketV1, ID: socketID, req: req, Connected: true}
		return v1.onConnect[socket.Namespace](socketV1)
	}
}

func doDisconnectPacket(v1 *ServerV1) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID siot.SocketID, socket siot.Socket) (err error) {
		v1.transport.Leave(socket.Namespace, socketID, socketIDPrefix+socketID.String())
		return nil
	}
}

func doEventPacket(v1 *ServerV1) func(siot.Socket) error {
	return func(socket siot.Socket) (err error) {
		switch data := socket.Data.(type) {
		case []interface{}:
			event, ok := data[0].(string)
			if !ok {
				return ErrBadEventName
			}

			if fn, ok := v1.events[socket.Namespace][event]; ok {
				return fn.Callback(data[1:]...)
			}
		case []string:
			event := data[0]
			if fn, ok := v1.events[socket.Namespace][event]; ok {
				err = fn.Callback(stoi(data[1:])...)
			}
		default:
			return ErrInvalidData.F(fmt.Sprintf("type %s", data)).KV("do", "eventPacket")
		}
		return err
	}
}

func doAckPacket(v1 *ServerV1) func(siot.Socket) error {
	return func(socket siot.Socket) (err error) {
		event := fmt.Sprintf("%s%d", ackIDEventPrefix, socket.AckID)
		switch data := socket.Data.(type) {
		case []interface{}:
			if fn, ok := v1.events[socket.Namespace][event]; ok {
				return fn.Callback(data...)
			}
		case []string:
			event := data[0]
			if fn, ok := v1.events[socket.Namespace][event]; ok {
				err = fn.Callback(stoi(data)...)
			}
		default:
			return ErrInvalidData.F(fmt.Sprintf("type %s", data)).KV("do", "ackPacket")
		}
		return err
	}
}
