package protocol

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ErrShortRead  errStr = "short read"
	ErrShortWrite errStr = "short write"

	ErrEmptyDataArray          errStr = "empty data array"
	ErrEmptyDataObject         errStr = "empty data object"
	ErrNoAckID                 errStr = "no ackID found"
	ErrPacketDataNotWritable   errStr = "no data io.writer found"
	ErrUnexpectedAttachmentEnd errStr = "unexpected attachment end"

	ErrUnexpectedJSONEnd errStr = "unexpected end of JSON input"
	ErrBadMarshal        errStr = "data marshal: %w"
	ErrBadUnmarshal      errStr = "data unmarshal: %w"
	ErrBadParse          errStr = "%s int parse: %w"

	ErrInvalidPacketType errStr = "the data packet type %T does not exist"
)

type PacketError struct {
	str    string
	buffer []byte

	errs []error
}

func (e PacketError) Error() string {
	if e.str != "" {
		return e.str
	}
	return e.errs[0].Error()
}

func (e PacketError) Is(target error) bool {
	for _, err := range e.errs {
		if ok := errors.Is(err, target); ok {
			return true
		}
	}
	return false
}

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

type readWriteErr struct{ error }

func (rw readWriteErr) Read([]byte) (int, error)  { return 0, rw.error }
func (rw readWriteErr) Write([]byte) (int, error) { return 0, rw.error }
