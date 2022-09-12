package session

import "time"

type sessionCtxKey string

const SessionTimeoutKey sessionCtxKey = "timeout"
const SessionIntervalKey sessionCtxKey = "interval"

type TimeoutChannel func() <-chan struct{}
type IntervalChannel func() <-chan time.Time
