package engineio

import (
	"net/http"
)

const Version4 EIOVersionStr = "4"

func init() { registery[Version4.Int()] = NewServerV4 }

// https://github.com/socketio/engine.io/tree/fe5d97fc3d7a26d34bce786a97962fae3d7ce17f
type serverV4 struct{ *serverV3 }

func NewServerV4(opts ...Option) Server { return (&serverV4{}).new(opts...) }

func (v4 *serverV4) new(opts ...Option) *serverV4 {
	v4.serverV3 = (&serverV3{}).new(opts...)
	v4.With(v4, opts...)
	return v4
}

func (v4 *serverV4) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	// codecs := transport.Codec{
	// 	PacketEncoder:  protocol.NewPacketEncoderV3, // v3 is the latest for EIO4...
	// 	PacketDecoder:  protocol.NewPacketDecoderV3,
	// 	PayloadEncoder: protocol.NewPayloadEncoderV4,
	// 	PayloadDecoder: protocol.NewPayloadDecoderV4,
	// }

	// esid := SessionID(r.URL.Query().Get("sid"))
	// name := transport.Name(r.URL.Query().Get("transport"))

	// if len(esid) == 0 {
	// 	esid := v4.newSessionID()
	// 	handshakePacket := protocol.Packet{
	// 		T: protocol.OpenPacket,
	// 		D: &protocol.HandshakeV3{ // v3 is the latest for EIO4...
	// 			HandshakeV2: protocol.HandshakeV2{
	// 				SID:         string(esid),
	// 				Upgrades:    v4.upgradeable(name, v4.transports),
	// 				PingTimeout: protocol.Duration(v4.pingTimeout),
	// 			},
	// 			PingInterval: protocol.Duration(v4.pingInterval),
	// 		},
	// 	}

	// 	return v4.handshake(esid, name, codecs, handshakePacket)(w, r)
	// }

	// serve, err := v4.sessions.Get(esid)
	// if err != nil {
	// 	return err
	// }

	// return serve.Run(func(o *runOption) { o.W = w; o.R = r })
	return nil
}
