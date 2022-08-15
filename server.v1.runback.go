package socketio

import (
	"fmt"

	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

// runV1 are the callbacks that are used for version 1 of the server based on the
// receive of the transport and the packet type. This can be different for the
// different server versions.
func runV1(v1 *ServerV1) func(*Request, SocketID) error {
	return func(r *Request, socketID SocketID) error {
		for socket := range v1.transport.Receive(socketID) {
			doV1(v1, r, socketID, socket)
		}
		return nil
	}
}

func doV1(v1 *ServerV1, r *Request, socketID SocketID, socket siot.Socket) {
	switch socket.Type {
	case siop.ConnectPacket.Byte():
		if err := v1.doConnectPacket(r, socketID, socket); err != nil {
			v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.DisconnectPacket.Byte():
		if err := v1.doDisconnectPacket(r, socketID, socket); err != nil {
			v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.EventPacket.Byte():
		if err := v1.doEventPacket(socketID, socket); err != nil {
			v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.AckPacket.Byte():
		if err := v1.doAckPacket(socketID, socket); err != nil {
			v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	default:
		err := ErrInvalidPacketType.F("v1", socket)
		v1.transport.Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
	}
}

// doConnectPacket the function
func doConnectPacket(v1 *ServerV1) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID SocketID, socket siot.Socket) (err error) {
		v1.transport.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		v1.setPrefix()
		v1.setSocketID(socketID)
		v1.setNsp(socket.Namespace)

		socketV1 := &SocketV1{inSocketV1: v1.inSocketV1, req: req, Connected: true}
		if fn, ok := v1.onConnect[socket.Namespace]; ok {
			return fn(socketV1)
		}
		return ErrBadOnConnectSocket
	}
}

func doDisconnectPacket(v1 *ServerV1) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID siot.SocketID, socket siot.Socket) (err error) {
		if fn, ok := v1.events[socket.Namespace][OnDisconnectEvent][socketID]; ok {
			v1.transport.Leave(socket.Namespace, socketID, socketIDPrefix+socketID.String())
			return fn.Callback("client namespace disconnect")
		}
		return ErrBadOnDisconnectSocket
	}
}

func doEventPacket(v1 *ServerV1) func(SocketID, siot.Socket) error {
	return func(socketID SocketID, socket siot.Socket) (err error) {
		switch data := socket.Data.(type) {
		case []interface{}:
			event, ok := data[0].(string)
			if !ok {
				return ErrBadEventName
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
				return fn.Callback(stoi(data[1:])...)
			}
			if fn, ok := v1.events[socket.Namespace][event][serverEvent]; ok {
				return fn.Callback(stoi(data[1:])...)
			}
		}
		return ErrInvalidData.F(fmt.Sprintf("type %s", socket.Data)).KV("do", "eventPacket")
	}
}

func doAckPacket(v1 *ServerV1) func(SocketID, siot.Socket) error {
	return func(socketID SocketID, socket siot.Socket) (err error) {
		event := fmt.Sprintf("%s%d", ackIDEventPrefix, socket.AckID)
		switch data := socket.Data.(type) {
		case []interface{}:
			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				return fn.Callback(data...)
			}
		case []string:
			event := data[0]
			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				err = fn.Callback(stoi(data)...)
			}
		default:
			return ErrInvalidData.F(fmt.Sprintf("type %s", data)).KV("do", "ackPacket")
		}
		return err
	}
}
