package engineio

func WithMaxPayload(n int) Option {
	return func(svr Server) {
	ServerCheck:
		switch v := svr.(type) {
		case *serverV4:
			v.maxPayload = n
		case interface{ prev() Server }:
			svr = v.prev()
			goto ServerCheck
		}
	}
}
