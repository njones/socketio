//go:build gc || (eio_svr_v4 && eio_svr_v5)
// +build gc eio_svr_v4,eio_svr_v5

package engineio

import (
	"net/http"
	"strings"

	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
)

const Version4 EIOVersionStr = "4"

func init() { registry[Version4.Int()] = NewServerV4 }

// https://github.com/socketio/engine.io/tree/fe5d97fc3d7a26d34bce786a97962fae3d7ce17f
// https://github.com/socketio/engine.io/compare/3.5.x...4.1.x

type serverV4 struct {
	*serverV3

	maxPayload int
	UseEIO3    bool
}

func NewServerV4(opts ...Option) Server { return (&serverV4{}).new(opts...) }

func (v4 *serverV4) new(opts ...Option) *serverV4 {
	v4.serverV3 = (&serverV3{}).new(opts...)

	v4.maxPayload = 100000
	v4.maxHttpBufferSize = 1e7

	v4.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV3,
		PacketDecoder:  eiop.NewPacketDecoderV3,
		PayloadEncoder: eiop.NewPayloadEncoderV4,
		PayloadDecoder: eiop.NewPayloadDecoderV4,
	}

	v4.servers[Version4] = v4

	v4.With(v4, opts...)
	return v4
}

func (v4 *serverV4) prev() Server { return v4.serverV3 }

func (v4 *serverV4) serveTransport(w http.ResponseWriter, r *http.Request) (transport eiot.Transporter, err error) {
	if origin := r.Header.Get("Origin"); origin != "" {
		if strings.EqualFold(origin, r.URL.Hostname()) {
			w.Header().Set("Access-Control-Allow-Origin", r.URL.Host)
		}
	}
	if strings.ToUpper(r.Method) == "OPTIONS" {
		return nil, IOR
	}

	sessionID := sessionIDFrom(r)
	if sessionID == "" {
		sessionID = v4.generateID()
		transportName := transportNameFrom(r)
		transport = v4.transports[transportName](sessionID, v4.codec)
		if err := v4.sessions.Set(transport); err != nil {
			return nil, err
		}

		transport.Send(v4.handshakePacket(sessionID, transportName))
		if v4.initialPackets != nil {
			v4.initialPackets(transport, r)
		}

		if t, ok := transport.(interface {
			Write(http.ResponseWriter, *http.Request) error
		}); ok {
			transport.Send(eiop.Packet{T: eiop.NoopPacket, D: eiot.WriteClose{}})
			return transport, ToEOH(t.Write(w, r))
		}
	}

	var isUpgrade bool
	transport, isUpgrade, err = v4.doUpgrade(v4.sessions.Get(sessionID))(w, r)
	if err != nil {
		return nil, err
	}

	var opts []eiot.Option
	if isUpgrade {
		opts = []eiot.Option{eiot.WithIsUpgrade(isUpgrade)}
	}

	go func() { v4.transportRunError <- transport.Run(w, r, append(v4.eto, opts...)...) }()

	return
}

func (v4 *serverV4) handshakePacket(sessionID SessionID, transportName eiot.Name) eiop.Packet {
	packet := v4.serverV3.handshakePacket(sessionID, transportName)
	packet.D = &eiop.HandshakeV4{HandshakeV3: packet.D.(*eiop.HandshakeV3), MaxPayload: v4.maxPayload}
	return packet
}
