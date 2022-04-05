//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

type HandshakeV3 struct {
	HandshakeV2
	PingInterval Duration `json:"pingInterval"`
}
