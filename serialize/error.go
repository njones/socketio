package serialize

import (
	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrSerializableBinary erro.String = "can not serialize the object, use Read instead"
	ErrParseOutOfBounds   erro.String = "the parsed %s (size:%d) is out of bounds."
)
