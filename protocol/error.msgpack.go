package protocol

import erro "github.com/njones/socketio/internal/errors"

const (
	ErrDecodeBase64Failed erro.StringF = "failed to decode msgpack base64 field:: %w"
	ErrDecodeFieldFailed  erro.StringF = "failed to decode msgpack field:: %w"
	ErrEncodeFieldFailed  erro.StringF = "failed to encode msgpack field:: %w"
)
