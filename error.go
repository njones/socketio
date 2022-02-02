package socketio

import (
	"fmt"
	"strings"
)

const (
	ErrSerializeBinary   errStr = "can not serialize the object, instead %s"
	ErrUnserializeBinary errStr = "can not unserialize the object, instead %s"

	ErrBadScrub         errStr = "bad scrub to string: %w"
	ErrBadEventName     errStr = "bad event name: %s"
	ErrInvalidData      errStr = "invalid data type: %s"
	ErrInvalidEventName errStr = "invalid event name, cannot use the registered name %q"

	ErrStubSerialize   errStr = "no Serialize() is a callback function"
	ErrStubUnserialize errStr = "no Unserialize() is a callback function"

	ErrInvalidDataInParams errStr = "the data coming in is not the same as the passed in parameters"
	ErrInvalidFuncInParams errStr = "need pass in the same number of parameters as the passed in function"
	ErrSingleOutParam      errStr = "need to have a single error output for the passed in function"
	ErrBadParamType        errStr = "bad type for parameter"
	ErrInterfaceNotFound   errStr = "need to have interface for serialize"
	ErrUnknownPanic        errStr = "unknown panic"
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
