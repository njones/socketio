package engineio

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	eios "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
)

func loadDuration(addr *time.Duration) time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(addr)))
}

func storeDuration(addr *time.Duration, val time.Duration) {
	atomic.StoreInt64((*int64)(addr), int64(val))
}

type transportSessions interface {
	Set(eiot.Transporter) error
	Get(SessionID) (eiot.Transporter, error)

	WithCancel(ctx context.Context) context.Context
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
			if t, ok := tr.s[sessionID]; ok {
				t.SendTimeout()
			}
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

	return nil, ErrUnknownSessionID
}

type lifecycle struct {
	id, td, shave time.Duration

	t      *sync.Map
	i      *sync.Map
	cancel *sync.Map

	removeTransport func(SessionID)
}

func (c *lifecycle) WithCancel(ctx context.Context) context.Context {
	sessionID, ok := ctx.Value(ctxSessionID).(SessionID)
	if !ok {
		// there is no session to attach the timer to
		return ctx
	}

	var chanPrefix = "chan:done:"
	c.cancel.LoadOrStore(sessionID.PrefixID(chanPrefix), make(chan func(), 1))

	ctx = context.WithValue(ctx, eios.SessionCloseChannelKey, func() <-chan func() {
		if ch, ok := c.cancel.Load(sessionID.PrefixID(chanPrefix)); ok {
			return ch.(chan func())
		}
		return nil
	})

	// Cancel will wait for another connections to close before closing this connection.
	// As of now this requires all of the sessions to be on a single server, by using
	// sticky sessions, otherwise this may not work as expected.
	ctx = context.WithValue(ctx, eios.SessionCloseFunctionKey, func() func() {
		if fn, ok := c.cancel.Load(sessionID.PrefixID(chanPrefix)); ok {
			syn := new(sync.WaitGroup)
			syn.Add(1)
			fn.(chan func()) <- func() { syn.Done() }
			close(fn.(chan func()))
			syn.Wait()
			return func() {
				c.removeSession(sessionID)
				if c.removeTransport != nil {
					c.removeTransport(sessionID)
				}
				c.cancel.Delete(sessionID.PrefixID(chanPrefix))
			}
		}
		return nil
	})
	return ctx
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

	storeDuration(&c.id, d)
	c.i.LoadOrStore(sessionID, time.NewTicker(c.id))

	var interval eios.IntervalChannel = func() <-chan time.Time {
		if val, ok := c.i.Load(sessionID); ok {
			val.(*time.Ticker).Reset(loadDuration(&c.id))
			return val.(*time.Ticker).C
		}
		return nil
	}

	return context.WithValue(ctx, eios.SessionIntervalKey, interval)
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

	storeDuration(&c.td, d)
	if val, ok := c.t.Load(sessionID); ok {
		reset := (loadDuration(&c.td) + loadDuration(&c.id)) - loadDuration(&c.shave)
		val.(*time.Timer).Stop()
		val.(*time.Timer).Reset(reset)
	} else {
		reset := (loadDuration(&c.td) + loadDuration(&c.id)) - loadDuration(&c.shave)
		c.t.Store(sessionID, time.NewTimer(reset))
		c.setTimeout(sessionID, time.Now())
	}

	x, cancel := context.WithCancel(ctx)
	c.cancel.Store(sessionID, func() { cancel() })

	var timeout eios.TimeoutChannel = func() <-chan struct{} {
		return x.Done()
	}

	x = context.WithValue(x, eios.SessionExtendTimeoutKey, eios.ExtendTimeoutFunc(func() {
		if val, ok := c.t.Load(sessionID); ok {
			reset := (loadDuration(&c.td) + loadDuration(&c.id)) - loadDuration(&c.shave)
			val.(*time.Timer).Stop()
			val.(*time.Timer).Reset(reset)
		}
	}))

	return context.WithValue(x, eios.SessionTimeoutKey, timeout)
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
		c.cancel.Delete(sessionID)
	}()
}

func (c *lifecycle) removeSession(sessionID SessionID) {
	c.t.Delete(sessionID)
	c.i.Delete(sessionID)
}
