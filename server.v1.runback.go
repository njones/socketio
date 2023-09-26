package socketio

import (
	"errors"
	"fmt"
	"net/http"

	eiot "github.com/njones/socketio/engineio/transport"
	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func autoConnect(v1 *ServerV1) func(transport eiot.Transporter, r *http.Request) {
	return func(transport eiot.Transporter, r *http.Request) {
		socketID, err := v1.tr().Add(transport)
		if err != nil {
			v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
			return
		}

		socket := siot.Socket{
			Type:      siop.ConnectPacket.Byte(),
			Namespace: "/",
		}

		if err := v1.doConnectPacket(socketID, socket, sioRequest(r)); err != nil {
			v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
			return
		}
	}
}

// runV1 are the callbacks that are used for version 1 of the server based on the
// receive of the transport and the packet type. This can be different for the
// different server versions.
func runV1(v1 *ServerV1) func(SocketID, *Request) error {
	return func(socketID SocketID, req *Request) error {
		for socket := range v1.tr().Receive(socketID) {
			if err := doV1(v1, socketID, socket, req); err != nil {
				return err
			}
		}
		return nil
	}
}

func doV1(v1 *ServerV1, socketID SocketID, socket siot.Socket, req *Request) error {
	switch socket.Type {
	case siop.ConnectPacket.Byte():
		if err := v1.doConnectPacket(socketID, socket, req); err != nil {
			v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.DisconnectPacket.Byte():
		if err := v1.doDisconnectPacket(socketID, socket, req); err != nil {
			if errors.Is(err, ErrOnDisconnectSocket) {
				return nil
			}
			v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.EventPacket.Byte():
		if err := v1.doEventPacket(socketID, socket); err != nil {
			v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.AckPacket.Byte():
		if err := v1.doAckPacket(socketID, socket); err != nil {
			v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
		}
	case siop.ErrorPacket.Byte():
		if e, ok := socket.Data.(error); ok {
			return e
		}
	default:
		err := ErrUnexpectedPacketType.F(socket).KV(ver, "v1")
		v1.tr().Send(socketID, serviceError(err), siop.WithType(siop.ErrorPacket.Byte()))
	}
	return nil
}

// doConnectPacket the function
func doConnectPacket(v1 *ServerV1) func(SocketID, siot.Socket, *Request) error {
	return func(socketID SocketID, socket siot.Socket, req *Request) (err error) {
		unlock := v1.r()
		tr := v1.tr()
		unlock()

		tr.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		v1.setPrefix()
		v1.setSocketID(socketID)
		v1.setNsp(socket.Namespace)

		if fn, ok := v1.onConnect[socket.Namespace]; ok {
			tr.Send(socketID, nil, siop.WithType(byte(siop.ConnectPacket)), siop.WithNamespace(socket.Namespace))
			return fn(&SocketV1{inSocketV1: v1.inSocketV1.clone(), req: req, Connected: true})
		}
		return ErrOnConnectSocket
	}
}

func doDisconnectPacket(v1 *ServerV1) func(SocketID, siot.Socket, *Request) error {
	return func(socketID SocketID, socket siot.Socket, req *Request) (err error) {
		if fn, ok := v1.events[socket.Namespace][OnDisconnectEvent][socketID]; ok {
			v1.tr().Leave(socket.Namespace, socketID, socketIDPrefix+socketID.String())
			v1.tr().Disconnect(socketID)
			return fn.Callback("client namespace disconnect")
		}
		// for any socket id at the io. (server) level...
		if fn, ok := v1.events[socket.Namespace][OnDisconnectEvent][serverEvent]; ok {
			v1.tr().Leave(socket.Namespace, socketID, socketIDPrefix+socketID.String())
			v1.tr().Disconnect(socketID)
			return fn.Callback("client namespace disconnect")
		}
		return ErrOnDisconnectSocket
	}
}

func doEventPacket(v1 *ServerV1) func(SocketID, siot.Socket) error {
	return func(socketID SocketID, socket siot.Socket) (err error) {
		type callbackAck interface {
			CallbackAck(...interface{}) []interface{}
		}

		switch data := socket.Data.(type) {
		case []interface{}:
			event, ok := data[0].(string)
			if !ok {
				return ErrUnknownEventName
			}
			if socket.Type != siop.DisconnectPacket.Byte() {
				data = data[1:]
			}

			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				if socket.AckID > 0 {
					if fn, ok := fn.(callbackAck); ok {
						vals := fn.CallbackAck(data...)
						return v1.tr().Send(socketID, vals, siop.WithNamespace(socket.Namespace), siop.WithAckID(socket.AckID), siop.WithType(byte(siop.AckPacket)))
					}
				}
				return fn.Callback(data...)
			}
			if fn, ok := v1.events[socket.Namespace][event][serverEvent]; ok {
				if socket.AckID > 0 {
					if fn, ok := fn.(callbackAck); ok {
						vals := fn.CallbackAck(data...)
						return v1.tr().Send(socketID, vals, siop.WithNamespace(socket.Namespace), siop.WithAckID(socket.AckID), siop.WithType(byte(siop.AckPacket)))
					}
				}
				return fn.Callback(data...)
			}
		case []string:
			event := data[0]
			if socket.Type != siop.DisconnectPacket.Byte() {
				data = data[1:]
			}

			if fn, ok := v1.events[socket.Namespace][event][socketID]; ok {
				return fn.Callback(stoi(data)...)
			}
			if fn, ok := v1.events[socket.Namespace][event][serverEvent]; ok {
				return fn.Callback(stoi(data)...)
			}
		}
		return ErrUnexpectedData.F(socket.Data).KV("do", "eventPacket")
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
			return ErrUnexpectedData.F(data).KV("do", "ackPacket")
		}
		return err
	}
}
