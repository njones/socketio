package protocol

import (
	"errors"
	"fmt"

	errs "github.com/njones/socketio/internal/errors"
)

const (
	ErrOnReadSoBuffer errsPacket = "error reading packet type %s, save for buffer"

	ErrShortRead  errs.String = "short read"
	ErrShortWrite errs.String = "short write"

	ErrEmptyDataArray          errs.String = "empty data array"
	ErrEmptyDataObject         errs.String = "empty data object"
	ErrNoAckID                 errs.String = "no ackID found"
	ErrPacketDataNotWritable   errs.String = "no data io.writer found"
	ErrUnexpectedAttachmentEnd errs.String = "unexpected attachment end"

	ErrUnexpectedJSONEnd errs.String = "unexpected end of JSON input"
	ErrBadMarshal        errs.String = "data marshal: %w"
	ErrBadUnmarshal      errs.String = "data unmarshal: %w"
	ErrBadParse          errs.String = "%s int parse: %w"

	ErrInvalidPacketType errs.String = "the data packet type %T does not exist"
)

type errsPacket string

func (e errsPacket) Error() string { return string(e) }

func (e errsPacket) BufferF(kind string, buf []byte, errs ...error) PacketError {
	return PacketError{
		buffer: buf,
		errs:   append([]error{fmt.Errorf(string(e), kind)}, errs...),
	}
}

type PacketError struct {
	buffer []byte
	errs   []error
}

func (e PacketError) Error() string { return e.errs[0].Error() }

func (e PacketError) Is(target error) bool {
	for _, err := range e.errs {
		if ok := errors.Is(err, target); ok {
			return true
		}
	}
	return false
}

type readWriteErr struct{ error }

func (rw readWriteErr) Read([]byte) (int, error)  { return 0, rw.error }
func (rw readWriteErr) Write([]byte) (int, error) { return 0, rw.error }
