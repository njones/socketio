package protocol

// Option is a type that is used for configuring an existing Packet types with data
type Option func(Packet)

// WithType sets the Type field in a Packet object
func WithType(_type byte) Option {
	return func(packet Packet) { packet.WithType(_type) }
}

// WithNamespace sets the Namespace field in a Packet object
func WithNamespace(namespace string) Option {
	return func(packet Packet) { packet.WithNamespace(namespace) }
}

// WithAckID sets the AckID field in a Packet object
func WithAckID(ackID uint64) Option {
	return func(packet Packet) { packet.WithAckID(ackID) }
}
