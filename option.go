package socketio

type Option = func(Server)

// withOption is for all servers that allow the With(...Option) to be called externally
type withOption interface {
	With(Server, ...Option)
}
