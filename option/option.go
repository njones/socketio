package option

type Option func(OptionWith)
type OptionWith interface{ With(...Option) }
