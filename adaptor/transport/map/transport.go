package tmap

import (
	"sync"
	"sync/atomic"

	eiot "github.com/njones/socketio/engineio/transport"
	siop "github.com/njones/socketio/protocol"
	sess "github.com/njones/socketio/session"
	siot "github.com/njones/socketio/transport"
)

type (
	SessionID = eiot.SessionID
	SocketID  = sess.ID

	Option = siop.Option
	Socket = siot.Socket
	Data   = siot.Data

	Namespace = string
	Room      = string
)

// mapTransport is the structure that holds a mapping of all connected
// clients in memory.
type mapTransport struct {
	ackCount uint64

	// hold the eioSessionID to socketID relationship
	ṁ *sync.RWMutex
	m map[SessionID]SocketID

	// hold the socketID to transport relationship
	ṡ *sync.RWMutex
	s map[SocketID]*siot.Transport

	// hold the namespace/socketID to room relationship
	ṙ *sync.Mutex
	r map[Namespace]map[SocketID]map[Room]struct{}

	f siop.NewPacket
}

// NewMapTransport returns a mapTransport object with all defaults.
func NewMapTransport(sioUsePacketVersion siop.NewPacket) *mapTransport {
	return &mapTransport{
		ṁ: new(sync.RWMutex),
		m: make(map[SessionID]SocketID),
		ṡ: new(sync.RWMutex),
		s: make(map[SocketID]*siot.Transport),
		ṙ: new(sync.Mutex),
		r: make(map[Namespace]map[SocketID]map[Room]struct{}),
		f: sioUsePacketVersion,
	}
}

// AckID returns a new Ack Id based on an incrementing number.
// This number starts from one every time the server is restarted. This
// could cause issues if the server is restarting before the AckID that
// was previously sent is consumed.
func (tr *mapTransport) AckID() uint64 {
	atomic.AddUint64(&tr.ackCount, 1)
	return atomic.LoadUint64(&tr.ackCount)
}

func (tr *mapTransport) Transport(socketID SocketID) *siot.Transport {
	tr.ṡ.Lock()
	defer tr.ṡ.Unlock()
	return tr.s[socketID]
}

// socketID to transport relationship methods

// Add creates a new socket id based on adding the EngineIO transport
// to the internal map. It returns the new socket id and any errors.
func (tr *mapTransport) Add(et eiot.Transporter) (SocketID, error) {
	sessionID := et.ID()

	tr.ṁ.Lock()
	if _, ok := tr.m[sessionID]; !ok {
		tr.m[sessionID] = sess.GenerateID()
	}
	socketID := tr.m[et.ID()]
	tr.ṁ.Unlock()

	return socketID, tr.Set(socketID, et)
}

func (tr *mapTransport) Set(socketID SocketID, et eiot.Transporter) error {
	tr.ṡ.Lock()
	defer tr.ṡ.Unlock()

	if et == nil {
		return ErrNilTransporter
	}

	tr.s[socketID] = siot.NewTransport(socketID, et, tr.f)
	return nil
}

func (tr *mapTransport) Receive(socketID SocketID) <-chan Socket {
	tr.ṡ.Lock()
	defer tr.ṡ.Unlock()

	if _, ok := tr.s[socketID]; ok {
		return tr.s[socketID].Receive()
	}
	return nil
}

func (tr *mapTransport) Send(socketID SocketID, data Data, opts ...Option) error {
	tr.ṡ.Lock()
	defer tr.ṡ.Unlock()

	if _, ok := tr.s[socketID]; ok {
		tr.s[socketID].Send(data, opts...)
		return nil
	}
	return ErrInvalidSocketTransport.F("map")
}

// namespace/socketID to room relationship

func (tr *mapTransport) Join(ns Namespace, socketID SocketID, room Room) error {
	tr.ṙ.Lock()
	defer tr.ṙ.Unlock()

	if _, ok := tr.r[ns]; !ok {
		tr.r[ns] = make(map[SocketID]map[Room]struct{})
	}
	if _, ok := tr.r[ns][socketID]; !ok {
		tr.r[ns][socketID] = make(map[Room]struct{})
	}
	tr.r[ns][socketID][room] = struct{}{}
	return nil
}

func (tr *mapTransport) Leave(ns Namespace, socketID SocketID, room Room) error {
	tr.ṙ.Lock()
	defer tr.ṙ.Unlock()

	if _, ok := tr.r[ns]; !ok {
		return nil
	}
	if _, ok := tr.r[ns][socketID]; !ok {
		return nil
	}

	delete(tr.r[ns][socketID], room)
	return nil
}

func (tr *mapTransport) Sockets(namespace Namespace) siot.SocketArray {
	var ids []SocketID
	for ns, socketIDs := range tr.r {
		if ns == namespace {
			for socketID := range socketIDs {
				ids = append(ids, socketID)
			}
		}
	}

	return siot.InitSocketArray(namespace, ids, siot.WithSocketRoomFilter(
		func(ns Namespace, rm Room, id SocketID) (bool, error) {
			if _ns, ok := tr.r[ns]; ok {
				if _id, ok := _ns[id]; ok {
					if _, ok := _id[rm]; ok {
						return true, nil
					}
				}
			}
			return false, nil
		},
	))
}

func (tr *mapTransport) Rooms(namespace Namespace, socketID SocketID) siot.RoomArray {
	var names []Room

FindingRoomNames:
	for ns, sockets := range tr.r {
		if ns == namespace {
			for sID, rooms := range sockets {
				if sID == socketID {
					for rm := range rooms {
						names = append(names, rm)
					}
					break FindingRoomNames
				}
			}
		}
	}
	return siot.RoomArray{Rooms: names}
}
