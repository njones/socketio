package engineio

func WithMaxPayload(n int) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV4); ok {
			v.maxPayload = n
		}
	}
}
