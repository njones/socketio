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

func WithIsUpgrade(b bool) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case *WebsocketTransport:
			v.isUpgrade = b
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
