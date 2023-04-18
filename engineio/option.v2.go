package engineio

import (
	"net/http"
	"sync"
	"time"

	eiot "github.com/njones/socketio/engineio/transport"
)

func init() {
	withPath[1] = func(path string) Option {
		return func(o OptionWith) {
			if v, ok := o.(*serverV2); ok {
				v.path = &path
			}
		}
	}
}

func WithCodec(codec eiot.Codec) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.codec = codec
		}
	}
}

func WithPingTimeout(d time.Duration) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.pingTimeout = d
		}
	}
}

func WithUpgradeTimeout(d time.Duration) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.upgradeTimeout = d
		}
	}
}

func WithCookie(name, path string, httpOnly bool) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.cookie.name = name
			v.cookie.path = path
			v.cookie.httpOnly = httpOnly
		}
	}
}

func WithGenerateIDFunc(fn func() SessionID) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.generateID = fn
		}
	}
}

func WithTransportChannelBuffer(n int) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.transportChanBuf = n
		}
	}
}

func WithTransportOption(opts ...eiot.Option) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.eto = append(v.eto, opts...)
		}
	}
}

var clearTransport = new(sync.Once)

func WithTransport(name eiot.Name, tr func(SessionID, eiot.Codec) eiot.Transporter) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			clearTransport.Do(func() {
				v.transports = make(map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter)
			})
			v.transports[name] = tr
		}
	}
}

func WithInitialPackets(fn func(eiot.Transporter, *http.Request)) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.initialPackets = fn
		}
	}
}

func WithSessionShave(d time.Duration) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV2); ok {
			v.sessions.(*sessions).shave = d
		}
	}
}
