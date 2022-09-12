package session

import (
	"crypto/rand"
	"encoding/base64"
)

type ID string

func (id ID) String() string            { return string(id) }
func (id ID) PrefixID(prefix string) ID { return ID(prefix + string(id)) }

var GenerateID = func() ID {
	b := make([]byte, 16)
	rand.Read(b)

	return ID("eio-" + base64.RawURLEncoding.EncodeToString(b))
}
