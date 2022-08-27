package transport

import "time"

type Option func(Transporter)

func WithCodec(codec Codec) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case interface{ Transport() *Transport }:
			v.Transport().codec = codec
		}
	}
}

func WithPingTimeout(dur time.Duration) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case interface{ Transport() *Transport }:
			v.Transport().pingTimeout = dur
		}
	}
}

func WithPingInterval(dur time.Duration) Option {
	return func(t Transporter) {
		switch v := t.(type) {
		case interface{ Transport() *Transport }:
			v.Transport().pingInterval = dur
		}
	}
}
