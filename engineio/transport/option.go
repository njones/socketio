package transport

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
