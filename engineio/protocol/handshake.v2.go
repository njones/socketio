//go:build gc || (eio_pac_v2 && eio_pac_v3)
// +build gc eio_pac_v2,eio_pac_v3

package protocol

import (
	"math"
	"time"
)

type HandshakeV2 struct {
	SID         string   `json:"sid"`
	Upgrades    []string `json:"upgrades"`
	PingTimeout Duration `json:"pingTimeout"`
}

func (h *HandshakeV2) Len() int {
	var n int
	if h == nil {
		h = new(HandshakeV2)
	}

	n += emptyBracketsLength
	n += emptySIDLength
	n += commaLength
	n += emptyUpgradesLength
	n += commaLength
	n += pingTimeoutKeyLength
	n += 1 // for the next calculation even if 0

	// Now the data
	if h.PingTimeout > 0 {
		inSeconds := h.PingTimeout / Duration(time.Millisecond)
		if inSeconds > 0 {
			n += int(math.Floor(math.Log10(float64(inSeconds))))
		}
	}
	n += len(h.SID)
	for i, v := range h.Upgrades {
		if i > 0 {
			n += commaLength
		}
		n += emptyStringLength
		n += len(v)
	}
	return n
}
