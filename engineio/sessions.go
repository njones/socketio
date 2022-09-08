package engineio

import (
	"context"
	"sync"
	"time"

	eios "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
)

type mapSessions interface {
	Set(eiot.Transporter) error
	Get(SessionID) (eiot.Transporter, error)

	WithTimeout(ctx context.Context, d time.Duration) context.Context
	WithInterval(ctx context.Context, d time.Duration) context.Context
}

type sessions struct {
	*transport
	*lifecycle
}

func NewSessions() *sessions {
	tr := transport{
		ʘ: new(sync.RWMutex),
		s: make(map[SessionID]eiot.Transporter),
	}
	li := lifecycle{
		t:      new(sync.Map),
		i:      new(sync.Map),
		cancel: new(sync.Map),
		shave:  10 * time.Millisecond,
		removeTransport: func(sessionID SessionID) {
			tr.ʘ.Lock()
			delete(tr.s, sessionID)
			tr.ʘ.Unlock()
		},
	}
	return &sessions{transport: &tr, lifecycle: &li}
}

type transport struct {
	ʘ *sync.RWMutex
	s map[SessionID]eiot.Transporter
}

func (t *transport) Set(tr eiot.Transporter) error {
	t.ʘ.Lock()
	t.s[tr.ID()] = tr
	t.ʘ.Unlock()

	return nil
}

func (t *transport) Get(sessionID SessionID) (eiot.Transporter, error) {
	t.ʘ.RLock()
	defer t.ʘ.RUnlock()

	if tr, ok := t.s[sessionID]; ok {
		return tr, nil
	}

	return nil, ErrNoSessionID
}

type lifecycle struct {
	id, td, shave time.Duration

	t      *sync.Map
	i      *sync.Map
	cancel *sync.Map

	removeTransport func(SessionID)
}

func (c *lifecycle) WithTimeout(ctx context.Context, d time.Duration) context.Context {
	if d <= 0 {
		return ctx
	}

	sessionID, ok := ctx.Value(ctxSessionID).(SessionID)
	if !ok {
		// there is no session to attach the timer to
		return ctx
	}

	c.td = d
	if val, ok := c.t.Load(sessionID); ok {
		val.(*time.Timer).Stop()
		val.(*time.Timer).Reset((c.td + c.id) - c.shave)

		x, cancel := context.WithCancel(ctx)
		c.cancel.Store(sessionID, func() { cancel() })
		var timeout eios.TimeoutChannel = func() <-chan struct{} {
			return x.Done()
		}

		x = context.WithValue(x, eios.SessionExtendTimeoutKey, eios.ExtendTimeoutFunc(func() {
			if val, ok := c.t.Load(sessionID); ok {
				val.(*time.Timer).Stop()
				val.(*time.Timer).Reset((c.td + c.id) - c.shave)
			}
		}))

		return context.WithValue(x, eios.SessionTimeoutKey, timeout)
	}

	c.t.Store(sessionID, time.NewTimer((c.td+c.id)-c.shave))

	x, cancel := context.WithCancel(ctx)
	c.cancel.Store(sessionID, func() { cancel() })
	c.setTimeout(sessionID, time.Now())
	var timeout eios.TimeoutChannel = func() <-chan struct{} {
		return x.Done()
	}

	x = context.WithValue(x, eios.SessionExtendTimeoutKey, eios.ExtendTimeoutFunc(func() {
		if val, ok := c.t.Load(sessionID); ok {
			val.(*time.Timer).Stop()
			val.(*time.Timer).Reset((c.td + c.id) - c.shave)
		}
	}))
	return context.WithValue(x, eios.SessionTimeoutKey, timeout)
}

func (c *lifecycle) WithInterval(ctx context.Context, d time.Duration) context.Context {
	if d <= 0 {
		return ctx
	}

	sessionID, ok := ctx.Value(ctxSessionID).(SessionID)
	if !ok {
		// there is no session to attach the timer to
		return ctx
	}

	c.id = d
	if val, ok := c.i.Load(sessionID); ok {
		t := val.(*time.Ticker)
		t.Reset(c.id)

		var ticker eios.IntervalChannel = func() <-chan time.Time {
			val, _ := c.i.Load(sessionID)
			val.(*time.Ticker).Reset(c.id)
			return val.(*time.Ticker).C
		}
		return context.WithValue(ctx, eios.SessionIntervalKey, ticker)
	}

	if val, ok := c.t.Load(sessionID); ok {
		timer := val.(*time.Timer)
		timer.Stop()
		timer.Reset((c.td + c.id) - c.shave)
	}

	c.i.Store(sessionID, time.NewTicker(c.id))
	val, _ := c.i.Load(sessionID)
	val.(*time.Ticker).Stop()

	var interval eios.IntervalChannel = func() <-chan time.Time {
		val, _ := c.i.Load(sessionID)
		val.(*time.Ticker).Reset(c.id)
		return val.(*time.Ticker).C
	}

	return context.WithValue(ctx, eios.SessionIntervalKey, interval)
}

func (c *lifecycle) setTimeout(sessionID SessionID, start time.Time) {
	go func() {
		val, _ := c.t.Load(sessionID)
		<-val.(*time.Timer).C

		cancel, _ := c.cancel.Load(sessionID)
		cancel.(func())()

		c.removeSession(sessionID)
		if c.removeTransport != nil {
			c.removeTransport(sessionID)
		}
	}()
}

func (c *lifecycle) removeSession(sessionID SessionID) {
	c.t.Delete(sessionID)
	c.i.Delete(sessionID)
}
