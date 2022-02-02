package protocol

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ErrInvalidRune        errStr = "invalid rune"
	ErrInvalidPacketType  errStr = "invalid packet type: %s"
	ErrInvalidPacketData  errStr = "invalid packet data: %s"
	ErrInvalidHandshake   errStr = "[%s] invalid handshake data"
	ErrHandshakeDecode    errStr = "[%s] handshake decode: %w"
	ErrHandshakeEncode    errStr = "[%s] handshake encode: %w"
	ErrPacketDecode       errStr = "[%s] packet decode: %w"
	ErrPacketEncode       errStr = "[%s] packet encode: %w"
	ErrPayloadDecode      errStr = "[%s] payload decode: %w"
	ErrPayloadEncode      errStr = "[%s] payload encode: %w"
	ErrBuffReaderRequired errStr = "please use a *bufio.Reader"
)

type (
	errStr    string
	errStruct struct {
		e, rr error
		wrap  []error
		f     [][]interface{}
		kv    []interface{}
	}
)

func (e errStr) Error() string { return string(e) }

func (e errStr) F(v ...interface{}) errStruct {
	var (
		f  [][]interface{}
		kv []interface{}
	)
	for i, val := range v {
		switch value := val.(type) {
		case errStruct:
			if len(value.f) > 0 {
				f = append(f, value.f...)
			}
			if len(value.kv) > 0 {
				kv = append(kv, value.kv...)
				value.kv = nil
				v[i] = value
			}
		}
	}

	var errs []error
	err := fmt.Errorf(string(e), v...)
	for une := errors.Unwrap(err); une != nil; une = errors.Unwrap(une) {
		errs = append(errs, une)
	}

	return errStruct{e: e, rr: err, f: f, kv: kv, wrap: errs}
}

func (e errStr) KV(kv ...interface{}) errStruct {
	return errStruct{e: e, rr: e, kv: kv}
}

func (e errStruct) Error() string {
	str := e.rr.Error() + fmtKV(e.kv)
	for _, v := range e.f {
		if len(v) > 0 {
			str = fmt.Sprintf(str, v...)
		}
	}
	return str
}

func (e errStruct) F(v ...interface{}) errStruct {
	return errStruct{e: e.e, rr: e.rr, kv: e.kv, f: append(e.f, v)}
}

func (e errStruct) KV(kv ...interface{}) errStruct {
	return errStruct{e: e.e, rr: e.rr, kv: append(e.kv, kv...)}
}

func (e errStruct) Is(target error) bool {
	if eStruct, ok := target.(errStruct); ok {
		return e.e.Error() == eStruct.e.Error()
	}
	if eStr, ok := target.(errStr); ok {
		return e.e.Error() == eStr.Error()
	}
	for _, werr := range e.wrap {
		if errors.Is(werr, target) {
			return true
		}
	}
	return false
}

func fmtKV(kvPairs []interface{}) string {
	if len(kvPairs) == 0 {
		return ""
	}

	pairs := make([]string, len(kvPairs)/2)
	for i, n := 0, 0; n < len(kvPairs); i, n = i+1, n+2 {
		key, val := kvPairs[n], interface{}("")
		if n+1 < len(kvPairs) {
			val = kvPairs[n+1]
		}
		pairs[i] = fmt.Sprint(key, `":"`, val)
	}

	return fmt.Sprintf("\t"+`{"%s"}`, strings.Join(pairs, `","`))
}
