package socketio

import (
	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrBadScrub                  erro.String = "bad scrub to string: %w"
	ErrBadEventName              erro.String = "bad event name: %s"
	ErrInvalidData               erro.String = "invalid data type: %s"
	ErrInvalidEventName          erro.String = "invalid event name, cannot use the registered name %q"
	ErrInvalidPacketType         erro.String = "invalid %s packet type: %#v"
	ErrInvalidPacketTypeExpected erro.String = "event packet invalid type: %T expected binary or string array"
	ErrNamespaceNotFound         erro.String = "namespace not found: %q"

	ErrStubSerialize   erro.String = "no Serialize() is a callback function"
	ErrStubUnserialize erro.String = "no Unserialize() is a callback function"

	ErrInvalidDataInParams erro.String = "the data coming in is not the same as the passed in parameters"
	ErrInvalidFuncInParams erro.String = "need pass in the same number of parameters as the passed in function"
	ErrSingleOutParam      erro.String = "need to have a single error output for the passed in function"
	ErrBadParamType        erro.String = "bad type for parameter"
	ErrInterfaceNotFound   erro.String = "need to have interface for serialize"
	ErrUnknownPanic        erro.String = "unknown panic"

	ErrOnBinaryEvent erro.String = "binary event: %v"

	ErrBadSendToSocketIndex  erro.String = "the index is invalid"
	ErrBadOnConnectSocket    erro.String = "bad onconnect socket"
	ErrBadOnDisconnectSocket erro.String = "bad ondisconnect socket"

	ErrFromRoom erro.String = "bad from room: %w"
)
