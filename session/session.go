package session

import (
	"crypto/rand"
	"encoding/base64"
)

type ID string

func (id ID) String() string            { return string(id) }
func (id ID) Room(prefix string) string { return prefix + string(id) }

var GenerateID = func() ID {
	b := make([]byte, 16)
	rand.Read(b)

	return ID("sio-" + base64.RawURLEncoding.EncodeToString(b))
}
