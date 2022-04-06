package callback

import (
	errs "github.com/njones/socketio/internal/errors"
)

const (
	ErrStubSerialize   errs.String = "no Serialize() is a callback function"
	ErrStubUnserialize errs.String = "no Unserialize() is a callback function"

	ErrInvalidDataInParams errs.String = "the data coming in is not the same as the passed in parameters"
	ErrInvalidFuncInParams errs.String = "need pass in the same number of parameters as the passed in function"
	ErrSingleOutParam      errs.String = "need to have a single error output for the passed in function"
	ErrBadParamType        errs.String = "bad type for parameter"
	ErrInterfaceNotFound   errs.String = "need to have interface for serialize"
	ErrUnknownPanic        errs.String = "unknown panic"
)
