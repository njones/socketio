package socketio

import (
	errs "github.com/njones/socketio/internal/errors"
)

const (
	ErrBadScrub          errs.String = "bad scrub to string: %w"
	ErrBadEventName      errs.String = "bad event name: %s"
	ErrInvalidData       errs.String = "invalid data type: %s"
	ErrInvalidEventName  errs.String = "invalid event name, cannot use the registered name %q"
	ErrInvalidPacketType errs.String = "invalid %s packet type: %#v"

	ErrStubSerialize   errs.String = "no Serialize() is a callback function"
	ErrStubUnserialize errs.String = "no Unserialize() is a callback function"

	ErrInvalidDataInParams errs.String = "the data coming in is not the same as the passed in parameters"
	ErrInvalidFuncInParams errs.String = "need pass in the same number of parameters as the passed in function"
	ErrSingleOutParam      errs.String = "need to have a single error output for the passed in function"
	ErrBadParamType        errs.String = "bad type for parameter"
	ErrInterfaceNotFound   errs.String = "need to have interface for serialize"
	ErrUnknownPanic        errs.String = "unknown panic"
)
