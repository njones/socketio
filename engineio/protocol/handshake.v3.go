//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import "math"

type HandshakeV3 struct {
	HandshakeV2
	PingInterval Duration `json:"pingInterval"`
}

const pingIntervalKeyLength = len(`"pingInterval":`)

func (h *HandshakeV3) Len() int {
	n := h.HandshakeV2.Len()
	n += commaLength
	n += pingIntervalKeyLength
	n += 1 // the (+1) for the next calculation: floor(log10()) + 1
	if h.PingInterval > 0 {
		n += int(math.Floor(math.Log10(float64(h.PingInterval))))
	}
	return n
}
