package engineio

type registar map[EIOVersionInt]func(opts ...Option) Server

func (m registar) latest(opts ...Option) Server {
	var ver EIOVersionInt
	for k := range m {
		if k > ver {
			ver = k
		}
	}
	return m[ver](opts...)
}

var registery = make(registar)
