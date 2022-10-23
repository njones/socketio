//go:build gc && !(sio_svr_v3 && sio_svr_v2 && sio_svr_v1)
// +build gc
// +build !sio_svr_v3 !sio_svr_v2 !sio_svr_v1

package socketio

func NewServer(opts ...Option) *ServerV4 {
	return NewServerV4(opts...)
}
