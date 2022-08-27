package protocol

import erro "github.com/njones/socketio/internal/errors"

const (
	ErrInvalidRune        erro.String = "invalid rune"
	ErrInvalidPacketType  erro.String = "invalid packet type: %s"
	ErrInvalidPacketData  erro.String = "invalid packet data: %s"
	ErrInvalidHandshake   erro.String = "[%s] invalid handshake data"
	ErrHandshakeDecode    erro.String = "[%s] handshake decode: %w"
	ErrHandshakeEncode    erro.String = "[%s] handshake encode: %w"
	ErrPacketDecode       erro.String = "[%s] packet decode: %w"
	ErrPacketEncode       erro.String = "[%s] packet encode: %w"
	ErrPayloadDecode      erro.String = "[%s] payload decode: %w"
	ErrPayloadEncode      erro.String = "[%s] payload encode: %w"
	ErrBuffReaderRequired erro.String = "please use a *bufio.Reader"

	EOR erro.String = "End Of Record: %w"
)
