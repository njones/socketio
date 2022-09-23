package socketio

import (
	"errors"
	"fmt"
	"time"

	siop "github.com/njones/socketio/protocol"
	siot "github.com/njones/socketio/transport"
)

func doConnectPacketV4(v4 *ServerV4) func(SocketID, siot.Socket, *Request) error {
	return func(socketID SocketID, socket siot.Socket, req *Request) (err error) {
		transport := v4.tr()
		transport.Join(socket.Namespace, socketID, socketID.Room(socketIDPrefix))

		stopBuffer := transport.(rawTransport).Transport(socketID).StartBuffer()
		defer stopBuffer()

		v4.setPrefix()
		v4.setSocketID(socketID)
		v4.setNsp(socket.Namespace)

		var h = handshakeV4{}
		switch val := socket.Data.(type) {
		case map[string]interface{}:
			h.Auth = func() map[string]interface{} {
				return val
			}
		}

		if fn, ok := v4.onConnect[socket.Namespace]; ok {
			return fn(&SocketV4{inSocketV4: v4.inSocketV4, req: req, han: h})
		}

		return ErrNamespaceNotFound.F(socket.Namespace)
	}
}

func runV4(v4 *ServerV4) func(SocketID, *Request) error {
	return func(socketID SocketID, req *Request) error {
		for socket := range v4.tr().Receive(socketID) {
			if err := doV4(v4, socketID, socket, req); err != nil {
				return err
			}
		}
		return nil
	}
}

func doV4(v4 *ServerV4, socketID SocketID, socket siot.Socket, req *Request) error {
	v1 := v4.prev.prev.prev

	switch socket.Type {
	case siop.ConnectPacket.Byte():
		if err := v1.doConnectPacket(socketID, socket, req); err != nil {
			if errors.Is(err, ErrNamespaceNotFound) {
				v4.tr().Send(socketID, serviceError(fmt.Errorf("%snvalid namespace", "I")), siop.WithNamespace(socket.Namespace), siop.WithType(byte(siop.ConnectErrorPacket)))
				return nil
			}
			v4.tr().Send(socketID, serviceError(err), siop.WithType(byte(siop.ConnectErrorPacket)))
			return nil
		}

		connectResponse := map[string]interface{}{"sid": socketID.String()}
		v4.tr().Send(socketID, connectResponse, siop.WithType(siop.ConnectPacket.Byte()), siop.WithNamespace(socket.Namespace))

		time.Sleep(600 * time.Microsecond) // wait, wait... what! wait for a 1/2 beat... otherwise it may ship things to fast to the websocket endpoint...

		var data = socket.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		connectNamespace := []interface{}{"auth", data}
		v4.tr().Send(socketID, connectNamespace, siop.WithType(siop.EventPacket.Byte()), siop.WithNamespace(socket.Namespace))

		time.Sleep(600 * time.Microsecond) // yup, things were flaky when there wasn't a sleep...

		v4.tr().(rawTransport).Transport(socketID).SendBuffer()
		return nil
	}
	return doV3(v4.prev, socketID, socket, req)
}
