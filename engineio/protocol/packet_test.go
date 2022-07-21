package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacketLength(t *testing.T) {
	runWithOption := func(opts ...testoption) func(Packet, int) func(*testing.T) {
		return func(data Packet, want int) func(*testing.T) {
			return func(*testing.T) {
				for _, opt := range opts {
					opt(t)
				}

				length := data.Len()
				assert.Equal(t, want, length)
			}
		}
	}

	var values = func(a Packet, b int) func() (Packet, int) {
		return func() (Packet, int) { return a, b }
	}

	tests := map[string]func() (data Packet, want int){
		"Open":              values(Packet{T: OpenPacket, D: new(HandshakeV2)}, 41),
		"Close":             values(Packet{T: ClosePacket}, 1),
		"Message with Text": values(Packet{T: MessagePacket, D: "HelloWorld"}, 11),
		"Ping with text":    values(Packet{T: PingPacket, D: "probe"}, 6),
		"Open with Handshake v2": values(Packet{T: OpenPacket, D: &HandshakeV2{
			SID:         "The ID here",
			Upgrades:    []string{},
			PingTimeout: 5000,
		}}, len(`0{"sid":"The ID here","upgrades":[],"pingTimeout":5000}`)),
		"Open with Handshake v3": values(Packet{T: OpenPacket, D: &HandshakeV3{
			HandshakeV2: HandshakeV2{
				SID:         "The ID here",
				Upgrades:    []string{},
				PingTimeout: 5000,
			},
			PingInterval: 5000,
		}}, len(`0{"sid":"The ID here","upgrades":[],"pingTimeout":5000,"pingInterval":5000}`)),
	}

	for name, testing := range tests {
		t.Run(name, runWithOption()(testing()))
	}
}
