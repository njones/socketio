package session

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

type ID string

func (id ID) String() string { return string(id) }
func (id ID) GoString() string {
	if b, a, found := strings.Cut(string(id), "::"); found {
		if str, err := hex.DecodeString(b); err == nil {
			return string(str) + "::" + a
		}
	}
	return string(id)
}
func (id ID) Room(prefix string) string { return prefix + string(id) }

var GenerateID = func(string) ID {
	b := make([]byte, 16)
	rand.Read(b)

	return ID("sio-" + base64.RawURLEncoding.EncodeToString(b))
}
