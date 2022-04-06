package serialize

import (
	errs "github.com/njones/socketio/internal/errors"
)

const (
	ErrSerializeBinary   errs.String = "can not serialize the object, instead %s"
	ErrUnserializeBinary errs.String = "can not unserialize the object, instead %s"
)
