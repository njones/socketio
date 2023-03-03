package protocol

import (
	"errors"
	"fmt"

	erro "github.com/njones/socketio/internal/errors"
)

// All of the errors that can happen while reading or writing socket.io packets
const (
	ErrReadFailed                  erro.StringF = "failed to read bytes:: %w"
	ErrUnmarshalInitialFieldFailed erro.StringF = "failed to unmarshal initial fields:: %w"
	ErrMarshalDataFailed           erro.StringF = "failed to marshal data:: %w"
	ErrUnmarshalDataFailed         erro.StringF = "failed to unmarshal data:: %w"
	ErrMarshalBinaryDataFailed     erro.StringF = "failed to marshal binary data:: %w"
	ErrUnmarshalBinaryDataFailed   erro.StringF = "failed to unmarshal binary data:: %w"
	ErrParseIntFailed              erro.StringF = "failed to parse int (%s):: %w"
	ErrUnexpectedPacketType        erro.StringF = "unexpected data type %T"
	ErrUnexpectedAttachmentEnd     erro.String  = "unexpected attachment end"
	ErrUnexpectedJSONEnd           erro.String  = "unexpected JSON end"
	ErrBinaryDataUnsupported       erro.String  = "binary data unsupported in this version"
	ErrReadUseBuffer               errsPacket   = "%s: read buffer"
	ErrShortRead                   erro.State   = "read: short"
	ErrShortWrite                  erro.State   = "write: short"
	ErrEmptyDataArray              erro.State   = "data array: empty"
)

// errsPacket is an error type that can send back PacketError errors.
// These errors contain a buffer of the data that was read or written so that short
// reads or writes can maintain the proper state.
type errsPacket string

// Error provides the method to allow this errsPacket type to be passed as an error
func (e errsPacket) Error() string { return string(e) }

// BufferF takes a string of kind that can be passed into the error string, it takes bytes buf
// as data that can be buffered, and a errs amount of the underlining errors that this error
// can be compared against using errors.Is.
func (e errsPacket) BufferF(kind string, buf []byte, errs ...error) PacketError {
	return PacketError{
		buffer: buf,
		errs:   append([]error{fmt.Errorf(string(e), kind)}, errs...),
	}
}

// A PacketError holds buffered data that can be used in the event of an error. The
// underlining error is still sent, and can be compared with using errors.Is.
type PacketError struct {
	buffer []byte
	errs   []error
}

// Error he method to allow this struct to be passed as an error
func (e PacketError) Error() string { return e.errs[0].Error() }

// Is matches the target error to one of the underlining errors
// attached to this struct.
func (e PacketError) Is(target error) bool {
	for _, err := range e.errs {
		if ok := errors.Is(err, target); ok {
			return true
		}
	}
	return false
}

// readWriteErr is a error that will be passed back as a reader/writer
// then the error is passed to the function that is reading or writing
// data to the reader/writer
type readWriteErr struct{ error }

// Read takes the internal error and passes it back to the caller
func (rw readWriteErr) Read([]byte) (int, error) { return 0, rw.error }

// Write takes the internal error and passes it back to the caller
func (rw readWriteErr) Write([]byte) (int, error) { return 0, rw.error }

func (rw readWriteErr) Error() string { return rw.error.Error() }
func (rw readWriteErr) Unwrap() error { return rw.error }
