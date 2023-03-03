package serialize

import (
	erro "github.com/njones/socketio/internal/errors"
)

const (
	ErrUnsupportedUseRead erro.String = "Serialize() method unsupported, use the Read() method instead"
	ErrUnsupported        erro.State  = "method: unsupported"
)
