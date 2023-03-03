package callback

import (
	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrUnimplementedSerialize   erro.String  = "unimplemented Serialize() method"
	ErrUnimplementedUnserialize erro.String  = "unimplemented Unserialize() method"
	ErrUnexpectedDataInParams   erro.StringF = "expected %d callback input parameters, found %d"
	ErrUnexpectedFuncInParams   erro.StringF = "expected %d wrap.Parameter values, found %d"
	ErrUnexpectedSingleOutParam erro.StringF = "expected a single error return parameter, found %d return parameters"
	ErrInterfaceNotFound        erro.String  = "interface not found for serialize"
	ErrUnknownPanic             erro.State   = "unknown panic"
)
