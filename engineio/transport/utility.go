package transport

type socketClose struct{ error }

func (sc socketClose) SocketCloseChannel() error { return sc.error }

type WriteClose struct{ error }

func (wc WriteClose) SocketCloseChannel() error { return wc.error }
