package session

import "time"

type sessionCtxKey string

const (
	SessionExtendTimeoutKey sessionCtxKey = "extendTimeout"
	SessionTimeoutKey       sessionCtxKey = "timeout"
	SessionIntervalKey      sessionCtxKey = "interval"
)

type ExtendTimeoutFunc func()
type TimeoutChannel func() <-chan struct{}
type IntervalChannel func() <-chan time.Time
