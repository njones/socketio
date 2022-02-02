package transport

type Option func(Transporter)

func WithCodec(codec Codec) Option {
	return func(t Transporter) {
		if tr, ok := t.(interface{ Transport() *Transport }); ok {
			tr.Transport().codec = codec
		}
	}
}
