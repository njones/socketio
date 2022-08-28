//go:build gc || eio_pac_v3
// +build gc eio_pac_v3

package protocol

import (
	"math"
)

type HandshakeV4 struct {
	*HandshakeV3
	MaxPayload int `json:"maxPayload"`
}

const maxPayloadKeyLength = len(`"maxPayload":`)

func (h *HandshakeV4) Len() int {
	n := h.HandshakeV3.Len()
	n += commaLength
	n += maxPayloadKeyLength
	n += 1 // the (+1) for the next calculation: floor(log10()) + 1
	if h.MaxPayload > 0 {
		n += int(math.Floor(math.Log10(float64(h.MaxPayload))))
	}
	return n
}
