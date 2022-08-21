package engineio

type registrar map[EIOVersionInt]func(opts ...Option) Server

func (m registrar) latest(opts ...Option) Server {
	if len(m) == 0 {
		return nil
	}

	if len(m) == 1 {
		for k := range m {
			return m[k](opts...)
		}
	}

	var ver EIOVersionInt
	for k := range m {
		if k > ver {
			ver = k
		}
	}

	return m[ver](opts...)
}

var registry = make(registrar)
