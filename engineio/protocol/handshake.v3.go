package protocol

type HandshakeV3 struct {
	HandshakeV2
	PingInterval Duration `json:"pingInterval"`
}
