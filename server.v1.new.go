//go:build sio_svr_v1
// +build sio_svr_v1

package socketio

// NewServer returns a new v1.0 socketIO server
func NewServer(opts ...Option) *ServerV1 {
	return NewServerV1(opts...)
}
