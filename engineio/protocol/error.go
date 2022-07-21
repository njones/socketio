package protocol

import errs "github.com/njones/socketio/internal/errors"

const (
	ErrInvalidRune        errs.String = "invalid rune"
	ErrInvalidPacketType  errs.String = "invalid packet type: %s"
	ErrInvalidPacketData  errs.String = "invalid packet data: %s"
	ErrInvalidHandshake   errs.String = "[%s] invalid handshake data"
	ErrHandshakeDecode    errs.String = "[%s] handshake decode: %w"
	ErrHandshakeEncode    errs.String = "[%s] handshake encode: %w"
	ErrPacketDecode       errs.String = "[%s] packet decode: %w"
	ErrPacketEncode       errs.String = "[%s] packet encode: %w"
	ErrPayloadDecode      errs.String = "[%s] payload decode: %w"
	ErrPayloadEncode      errs.String = "[%s] payload encode: %w"
	ErrBuffReaderRequired errs.String = "please use a *bufio.Reader"

	EOR errs.String = "End Of Record: %w"
)
