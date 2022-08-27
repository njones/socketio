package callback

import (
	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrStubSerialize   erro.String = "no Serialize() is a callback function"
	ErrStubUnserialize erro.String = "no Unserialize() is a callback function"

	ErrInvalidDataInParams erro.String = "the data coming in is not the same as the passed in parameters"
	ErrInvalidFuncInParams erro.String = "need pass in the same number of parameters as the passed in function"
	ErrSingleOutParam      erro.String = "need to have a single error output for the passed in function"
	ErrBadParamType        erro.String = "bad type for parameter"
	ErrInterfaceNotFound   erro.String = "need to have interface for serialize"
	ErrUnknownPanic        erro.String = "unknown panic"
)
