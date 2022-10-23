//go:build sio_svr_v2 && !sio_svr_v1
// +build sio_svr_v2,!sio_svr_v1

package socketio

func NewServer(opts ...Option) *ServerV2 {
	return NewServerV2(opts...)
}
