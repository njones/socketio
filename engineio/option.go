package engineio

import with "github.com/njones/socketio/internal/option"

type Option = with.Option
type OptionWith = with.OptionWith

// withPath stores all of the external options for WithPath
var withPath = make(with.OptionRegistry)

// WithPath could be accessed from SocketIO engine,
func WithPath(path string) Option {
	if opt, ok := withPath.Latest().(func(string) Option); ok {
		return opt(path)
	}
	return func(OptionWith) {}
}
