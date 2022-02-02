package protocol

type HandshakeV2 struct {
	SID         string   `json:"sid"`
	Upgrades    []string `json:"upgrades"`
	PingTimeout Duration `json:"pingTimeout"`
}
