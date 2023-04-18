package option

type Option func(OptionWith)
type OptionWith interface{ With(...Option) }
type OptionRegistry map[int]interface{}

func (r OptionRegistry) Latest() interface{} {
	if len(r) == 0 {
		return func(OptionWith) {}
	}
	var k, max int
	for k = range r {
		if k > max {
			max = k
		}
	}
	return r[k]
}
