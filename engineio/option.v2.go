package engineio

import (
	"net/http"
	"sync"
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
	ServerCheck:
		switch v := svr.(type) {
		case *serverV2:
			v.pingTimeout = d
		case interface{ prev() Server }:
			svr = v.prev()
			goto ServerCheck
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

func WithTransportChannelBuffer(n int) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV2:
			v.transportChanBuf = n
		}
	}
}

var clearTransports = new(sync.Once)

func WithTransport(name eiot.Name, tr func(SessionID, eiot.Codec) eiot.Transporter) Option {
	return func(svr Server) {
	ServerCheck: // makes things an O(2^n) check...
		switch v := svr.(type) {
		case *serverV2:
			clearTransports.Do(func() {
				v.transports = make(map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter)
			})
			v.transports[name] = tr
		case interface{ prev() Server }:
			svr = v.prev()
			goto ServerCheck
		}
	}
}

func WithInitialPackets(fn func(eiot.Transporter, *http.Request)) Option {
	return func(svr Server) {
	ServerCheck: // makes things an O(2^n) check...
		switch v := svr.(type) {
		case *serverV2:
			v.initialPackets = fn
		case interface{ prev() Server }:
			svr = v.prev()
			goto ServerCheck
		}
	}
}
