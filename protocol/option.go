package protocol

type Option func(Packet)

func WithType(_type byte) func(Packet) {
	return func(packet Packet) { packet.WithType(_type) }
}

func WithNamespace(namespace string) func(Packet) {
	return func(packet Packet) { packet.WithNamespace(namespace) }
}

func WithAckID(ackID uint64) func(Packet) {
	return func(packet Packet) { packet.WithAckID(ackID) }
}
