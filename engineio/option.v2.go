package engineio

import (
	"sync"
	"time"

	"github.com/njones/socketio/engineio/session"
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
			v.eto = append(v.eto, eiot.WithPingTimeout(d))
		}
	}
}

func WithUpgradeTimeout(d time.Duration) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.upgradeTimeout = d
		}
	}
}

var clearTransports = new(sync.Once)

func WithTransport(name eiot.Name, tr func(SessionID, eiot.Codec) eiot.Transporter) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			clearTransports.Do(func() {
				v.transports = make(map[eiot.Name]func(session.ID, eiot.Codec) eiot.Transporter)
			})
			v.transports[name] = tr
		}
	}
}

func WithCookie(name, path string, httpOnly bool) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.cookie.name = name
			v.cookie.path = path
			v.cookie.httpOnly = httpOnly
		}
	}
}

func WithGenerateIDFunc(fn func() SessionID) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.generateID = fn
		}
	}
}
