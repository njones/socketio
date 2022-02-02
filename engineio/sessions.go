package engineio

import (
	"sync"

	eiot "github.com/njones/socketio/engineio/transport"
)

type mapSessionToTransport interface {
	Set(eiot.Transporter) error
	Get(SessionID) (eiot.Transporter, error)
}

type sessionMap struct {
	ʘ *sync.RWMutex
	s map[SessionID]eiot.Transporter
}

func NewSessionMap() *sessionMap {
	return &sessionMap{ʘ: new(sync.RWMutex), s: make(map[SessionID]eiot.Transporter)}
}

func (m *sessionMap) Set(tr eiot.Transporter) error {
	m.ʘ.Lock()
	defer m.ʘ.Unlock()

	m.s[tr.ID()] = tr
	return nil
}

func (m *sessionMap) Get(sessionID SessionID) (eiot.Transporter, error) {
	m.ʘ.RLock()
	defer m.ʘ.RUnlock()

	if tr, ok := m.s[sessionID]; ok {
		return tr, nil
	}

	return nil, ErrNoSessionID.F("the sessionID to transport map")
}
