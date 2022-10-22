package engineio

import (
	"time"
)

func WithPingInterval(d time.Duration) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV3); ok {
			v.pingInterval = d
		}
	}
}

type (
	CORSenable               bool
	CORSorigin               []string
	CORSmethods              []string
	CORSheadersAllow         []string
	CORSheadersExpose        []string
	CORScredentials          bool
	CORSmaxAge               int
	CORSoptionsSuccessStatus int

	corsOption interface{ val() interface{} } // the value needs to be un-boxed
)

func (x CORSenable) val() interface{}               { return bool(x) }
func (x CORSorigin) val() interface{}               { return []string(x) }
func (x CORSmethods) val() interface{}              { return []string(x) }
func (x CORSheadersAllow) val() interface{}         { return []string(x) }
func (x CORSheadersExpose) val() interface{}        { return []string(x) }
func (x CORScredentials) val() interface{}          { return bool(x) }
func (x CORSmaxAge) val() interface{}               { return int(x) }
func (x CORSoptionsSuccessStatus) val() interface{} { return int(x) }

func WithCors(opts ...corsOption) Option {
	return func(o OptionWith) {
		if v, ok := o.(*serverV3); ok {
			for _, opt := range opts {
				switch opt.(type) {
				case CORSenable:
					v.cors.enable = opt.val().(bool)
				case CORSorigin:
					v.cors.origin = opt.val().([]string)
				case CORSmethods:
					v.cors.methods = opt.val().([]string)
				case CORSheadersAllow:
					v.cors.headersAllow = opt.val().([]string)
				case CORSheadersExpose:
					v.cors.headersExpose = opt.val().([]string)
				case CORScredentials:
					v.cors.credentials = opt.val().(bool)
				case CORSmaxAge:
					v.cors.maxAge = opt.val().(int)
				case CORSoptionsSuccessStatus:
					v.cors.optionsSuccessStatus = opt.val().(int)
				}
			}
		}
	}
}
