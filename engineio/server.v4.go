//go:build gc || (eio_svr_v4 && eio_svr_v5)
// +build gc eio_svr_v4,eio_svr_v5

package engineio

import (
	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
)

const Version4 EIOVersionStr = "4"

func init() { registery[Version4.Int()] = NewServerV4 }

// https://github.com/socketio/engine.io/tree/fe5d97fc3d7a26d34bce786a97962fae3d7ce17f
// https://github.com/socketio/engine.io/compare/3.5.x...4.1.x

type serverV4 struct {
	*serverV3

	UseEIO3 bool
}

func NewServerV4(opts ...Option) Server { return (&serverV4{}).new(opts...) }

func (v4 *serverV4) new(opts ...Option) *serverV4 {
	v4.serverV3 = (&serverV3{}).new(opts...)

	v4.maxHttpBufferSize = 1e7

	v4.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV3,
		PacketDecoder:  eiop.NewPacketDecoderV3,
		PayloadEncoder: eiop.NewPayloadEncoderV4,
		PayloadDecoder: eiop.NewPayloadDecoderV4,
	}

	v4.With(v4, opts...)
	return v4
}

func (v4 *serverV4) prev() Server { return v4.serverV3 }
