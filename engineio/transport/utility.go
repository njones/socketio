package transport

type socketClose struct{ error }

func (sc socketClose) SocketCloseChannel() error { return sc.error }
