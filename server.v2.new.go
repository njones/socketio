//go:build sio_svr_v2 && !sio_svr_v1
// +build sio_svr_v2,!sio_svr_v1

package socketio

func NewServer(opts ...Option) *ServerV2 {
	v2 := &ServerV2{}
	v2.new(opts...)
	return v2
}
