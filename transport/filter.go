package transport

import "bytes"

type SocketArray struct {
	namespace Namespace
	socketIDs []SocketID
	localIDs  [][]byte

	filterOnRoom    func(Namespace, Room, SocketID) (bool, error)
	filterToLocalID func(Namespace, SocketID) ([]byte, error)
}

func InitSocketArray(ns Namespace, ids []SocketID, opts ...func(*SocketArray)) SocketArray {
	array := SocketArray{
		namespace: ns,
		socketIDs: ids,
	}
	for _, opt := range opts {
		opt(&array)
	}
	return array
}

func (a SocketArray) IDs() []SocketID { return a.socketIDs }
func (a SocketArray) FromRoom(rm Room) (rtn []SocketID, err error) {
	for _, id := range a.socketIDs {
		if ok, _ := a.filterOnRoom(a.namespace, rm, id); ok {
			rtn = append(rtn, id)
		}
	}
	return rtn, nil
}
func (a SocketArray) LocalWith(id SocketID) (rtn []SocketID, err error) {
	lid, err := a.filterToLocalID(a.namespace, id)
	if err != nil {
		return nil, err
	}
	for i, id := range a.socketIDs {
		if bytes.Equal(lid, a.localIDs[i]) {
			rtn = append(rtn, id)
		}
	}
	return rtn, nil
}

type RoomArray struct {
	Rooms []Room
}

func (a RoomArray) Names() []Room { return a.Rooms }
