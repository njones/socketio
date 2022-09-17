package session

import "time"

type sessionCtxKey string

const (
	SessionTimeoutKey       sessionCtxKey = "timeout"
	SessionIntervalKey      sessionCtxKey = "interval"
	SessionExtendTimeoutKey sessionCtxKey = "timeout-extend"

	SessionCloseChannelKey  sessionCtxKey = "cancel-channel"
	SessionCloseFunctionKey sessionCtxKey = "cancel-function"
)

type (
	TimeoutChannel    func() <-chan struct{}
	IntervalChannel   func() <-chan time.Time
	ExtendTimeoutFunc func()
)
