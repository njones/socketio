package transport

import "time"

type Option func(Transporter)

func WithCodec(codec Codec) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case interface{ InnerTransport() *Transport }:
			v.InnerTransport().codec = codec
		}
	}
}

func OnInitProbe(b bool) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case *WebsocketTransport:
			v.isInitProbe = b
		}
	}
}

func OnUpgrade(fn func() error) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case *WebsocketTransport:
			v.fnOnUpgrade = fn
		}
	}
}

func WithNoPing() Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case interface{ InnerTransport() *Transport }:
			v.InnerTransport().sendPing = false
		}
	}
}

func WithBufferedReader() Option {
	return func(t Transporter) {
		switch v := t.(type) {
		// TODO(njones): case *PollingTransport: ...
		case *WebsocketTransport:
			v.buffered = true
		}
	}
}

func WithGovernor(minTime, sleep time.Duration) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case *WebsocketTransport:
			v.governor.minTime = minTime
			v.governor.sleep = sleep
		}
	}
}
