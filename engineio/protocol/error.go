package protocol

import erro "github.com/njones/socketio/internal/errors"

const ver = "version"

const (
	ErrUnexpectedPacketType  erro.StringF = "unexpected packet type %T"
	ErrUnexpectedPacketData  erro.StringF = "unexpected packet data %T"
	ErrUnexpectedHandshake   erro.StringF = "expected %s, found %T"
	ErrDecodeHandshakeFailed erro.StringF = "failed to decode handshake:: %w"
	ErrEncodeHandshakeFailed erro.StringF = "failed to encode handshake:: %w"
	ErrDecodePacketFailed    erro.StringF = "failed to decode packet:: %w"
	ErrEncodePacketFailed    erro.StringF = "failed to encode packet:: %w"
	ErrDecodePayloadFailed   erro.StringF = "failed to decode payload:: %w"
	ErrEncodePayloadFailed   erro.StringF = "failed to encode payload:: %w"
	EOR                      erro.StringF = "End Of Record: %w"
)

func kv(v ...interface{}) erro.Struct { return erro.KV(v...) }
