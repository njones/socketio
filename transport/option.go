package transport

type option func(*SocketArray)

func WithSocketRoomFilter(fn func(Namespace, Room, SocketID) (bool, error)) option {
	return func(ary *SocketArray) {
		ary.filterOnRoom = fn
	}
}
