//go:build gc || (eio_pac_v2 && eio_pac_v3)
// +build gc eio_pac_v2,eio_pac_v3

package protocol

type HandshakeV2 struct {
	SID         string   `json:"sid"`
	Upgrades    []string `json:"upgrades"`
	PingTimeout Duration `json:"pingTimeout"`
}
