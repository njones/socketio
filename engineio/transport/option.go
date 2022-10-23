package transport

import (
	"time"

	with "github.com/njones/socketio/option"
)

type Option = with.Option
type OptionWith = with.OptionWith

func WithCodec(codec Codec) Option {
	return func(o OptionWith) {
		switch v := o.(type) {
		case interface{ InnerTransport() *Transport }:
			v.InnerTransport().codec = codec
		}
	}
}

func OnInitProbe(b bool) Option {
	return func(o OptionWith) {
		if v, ok := o.(*WebsocketTransport); ok {
			v.isInitProbe = b
		}
	}
}

func OnUpgrade(fn func() error) Option {
	return func(o OptionWith) {
		if v, ok := o.(*WebsocketTransport); ok {
			v.fnOnUpgrade = fn
		}
	}
}

func WithNoPing() Option {
	return func(o OptionWith) {
		switch v := o.(type) {
		case interface{ InnerTransport() *Transport }:
			v.InnerTransport().sendPing = false
		}
	}
}

func WithBufferedReader() Option {
	return func(o OptionWith) {
		if v, ok := o.(*WebsocketTransport); ok {
			v.buffered = true
		}
	}
}

func WithGovernor(minTime, sleep time.Duration) Option {
	return func(o OptionWith) {
		if v, ok := o.(*WebsocketTransport); ok {
			v.governor.minTime = minTime
			v.governor.sleep = sleep
		}
	}
}
