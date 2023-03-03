package socketio

import (
	erro "github.com/njones/socketio/internal/errors"
)

const ver = "version"

const (
	ErrScrubFailed            erro.StringF = "failed to scrub string:: %w"
	ErrFromRoomFailed         erro.StringF = "failed to get socket ids from room:: %w"
	ErrUnknownEventName       erro.String  = "unknown event name, the first field is not a string"
	ErrUnknownBinaryEventName erro.StringF = "unknown event name, expected a string but found %v (%[1]T)"
	ErrUnsupportedEventName   erro.StringF = "event name unsupported, cannot use the registered name %q as an event name"
	ErrUnexpectedData         erro.StringF = "expected an []interface{} or []string, found %T"
	ErrUnexpectedBinaryData   erro.StringF = "expected an []interface{} (binary array) or []string, found %T"
	ErrUnexpectedPacketType   erro.StringF = "unexpected %T"
	ErrNamespaceNotFound      erro.StringF = "namespace %q not found"
	ErrOnConnectSocket        erro.State   = "socket: invalid onconnect"
	ErrOnDisconnectSocket     erro.State   = "socket: invalid ondisconnect"
)
