//go:build sio_svr_v1
// +build sio_svr_v1

package socketio

func NewServer(opts ...Option) *ServerV1 {
	v1 := &ServerV1{}
	v1.new(opts...)
	return v1
}
