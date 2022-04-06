package socketio

import (
	"fmt"

	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func runV2(v2 *ServerV2) func(r *Request, socketID siot.SocketID) error {
	return func(r *Request, socketID siot.SocketID) error {
		v1 := v2.prev

		for socket := range v1.transport.Receive(socketID) {
			switch socket.Type {
			case siop.ConnectPacket.Byte():
				if err := v1.doConnectPacket(r, socketID, socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
				}
			case siop.DisconnectPacket.Byte():
				return nil
			case siop.EventPacket.Byte():
				if err := v1.doEventPacket(socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
				}
			case siop.AckPacket.Byte():
				if err := v1.doAckPacket(socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
				}
			case siop.ErrorPacket.Byte():
				if e, ok := socket.Data.(error); ok {
					return e
				}
			case siop.BinaryEventPacket.Byte():
				if err := v2.doBinaryEventPacket(socket); err != nil {
					v1.transport.Send(socketID, serviceError(err), siop.WithType(byte(siop.ErrorPacket)))
				}
			}
		}

		return nil // should never reach here
	}
}

func doConnectPacketV2(v2 *ServerV2) func(*Request, SocketID, siot.Socket) error {
	return func(req *Request, socketID SocketID, socket siot.Socket) (err error) {
		v1 := v2.prev
		transport := v1.tr()
		transport.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		socketV2 := &SocketV2{ID: socketID, req: req}
		return v2.onConnect[socket.Namespace](socketV2)
	}
}

func doBinaryEventPacket(v2 *ServerV2) func(siot.Socket) error {
	return func(socket siot.Socket) (err error) {
		v1 := v2.prev

		switch data := socket.Data.(type) {
		case []interface{}:
			event, ok := data[0].(string)
			if !ok {
				return fmt.Errorf("binary event: %v", data[0])
			}
			if fn, ok := v1.events[socket.Namespace][event]; ok {
				err = fn.Callback(data[1:]...)
			}
		case []string:
			event := data[0]
			if fn, ok := v1.events[socket.Namespace][event]; ok {
				err = fn.Callback(stoi(data[1:])...)
			}
		default:
			return fmt.Errorf("event packet invalid type: %T expected binary or string array", socket.Data)
		}
		return err
	}
}
