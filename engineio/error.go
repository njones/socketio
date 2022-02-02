package engineio

import (
	"fmt"
	"strings"
)

const (
	EOH errStr = "end of hanshake"

	ErrBadPath        errStr = "bad URI path"
	ErrNoSessionID    errStr = "sessionID not found in: %s"
	ErrNoTransport    errStr = "transport not found"
	ErrOnTransportRun errStr = "bad transport run: %w"
	ErrPayloadEncode  errStr = "bad encode payload: %w"
)

type (
	errStr    string
	errStruct struct {
		e, rr error
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
	return errStruct{e: e, rr: fmt.Errorf(string(e), v...), f: f, kv: kv}
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
