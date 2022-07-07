//go:build !gc && !(sio_svr_v4 && sio_svr_v3 && sio_svr_v2 && sio_svr_v1)
// +build !gc
// +build !sio_svr_v4 !sio_svr_v3 !sio_svr_v2 !sio_svr_v1

package socketio

// NewServer returns a *ServerV4 which is a SocketIO server version 4. This is the default server.
// To have another server use the NewServer(V1) or build with the build tag sio_svr_v1.
func NewServer(opts ...Option) *ServerV4 {
	v4 := &ServerV4{}
	v4.new(opts...)
	return v4
}
