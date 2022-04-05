//go:build sio_svr_v3 && !(sio_svr_v2 && sio_svr_v1)
// +build sio_svr_v3
// +build !sio_svr_v2 !sio_svr_v1

package socketio

func NewServer(opts ...Option) *ServerV3 {
	v3 := &ServerV3{}
	v3.new(opts...)
	return v3
}
