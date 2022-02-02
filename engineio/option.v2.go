package engineio

import (
	"time"

	eiot "github.com/njones/socketio/engineio/transport"
)

type Option = func(Server)

func WithCodec(codec eiot.Codec) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.codec = codec
		}
	}
}

func WithPath(path string) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.path = &path
		}
	}
}

func WithPingTimeout(d time.Duration) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.pingTimeout = d
		}
	}
}

func WithTransport(name eiot.Name, tr func(SessionID, eiot.Codec) eiot.Transporter) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.transports[name] = tr
		}
	}
}

func WithGenerateID(fn func() SessionID) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.generateID = fn
		}
	}
}
