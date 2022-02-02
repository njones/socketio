package socketio

type Option = func(Server)

type withOption interface {
	With(Server, ...Option)
}
