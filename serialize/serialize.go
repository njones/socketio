package serialize

import (
	"encoding/json"
	"errors"
	"io"
	"math/bits"
	"strconv"
)

// https://github.com/socketio/socket.io/tree/master/examples/custom-parsers
type SerializableParam interface {
	Serializable
	Interface() interface{}
	Param() Serializable
}

type SerializableWrap interface {
	Serializable
	Interface() interface{}
}

type Serializable interface {
	Serialize() (string, error)
	Unserialize(string) error
}

func stringify(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case *_float64:
		if v != nil {
			return strconv.FormatFloat(float64(*v), 'f', -1, 64)
		}
	case *_int:
		if v != nil {
			return strconv.FormatInt(int64(*v), 10)
		}
	case *_string:
		if v != nil {
			return string(*v)
		}
	case *_uint:
		if v != nil {
			return strconv.FormatUint(uint64(*v), 10)
		}
	}
	return ""
}

var (
	AnyParam  = _anyWrap{Any(nil)}
	BinParam  = _binaryWrap{Binary(nil)}
	ErrParam  = _errorWrap{Error(nil)}
	F64Param  = _float64Param{Float64(0)}
	IntParam  = _intParam{Integer(0)}
	MapParam  = _mapParam{Map(nil)}
	StrParam  = _stringParam{String("")}
	UintParam = _uintParam{Uinteger(0)}
)

type (
	_any     struct{ a interface{} }
	_anyWrap struct{ SerializableWrap }
)

func Any(v interface{}) *_any                      { return &_any{v} }
func (x *_any) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_any) Serialize() (str string, err error) { return "", ErrUnsupported }
func (x *_any) Unserialize(str string) (err error) { return ErrUnsupported }
func (x *_any) Interface() (v interface{})         { return x.a }
func (x _anyWrap) Unserialize(string) error        { return nil }
func (x _anyWrap) String() string                  { return "" }

type (
	_binary     struct{ r io.Reader }
	_binaryWrap struct{ SerializableWrap }
)

func Binary(v io.Reader) *_binary                     { return &_binary{v} }
func (x *_binary) Read(p []byte) (n int, err error)   { return x.r.Read(p) }
func (x *_binary) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_binary) Serialize() (str string, err error) { return "", ErrUnsupportedUseRead }
func (x *_binary) Unserialize(str string) (err error) { return ErrUnsupported }
func (x *_binary) Interface() (v interface{})         { return x.r }
func (x _binaryWrap) Unserialize(string) error        { return nil }
func (x _binaryWrap) String() string                  { return "" }

type (
	_error     struct{ e error }
	_errorWrap struct{ SerializableWrap }
)

func Error(v error) *_error                          { return &_error{v} }
func (x *_error) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_error) Serialize() (str string, err error) { return x.e.Error(), nil }
func (x *_error) Unserialize(str string) (err error) { x.e = errors.New(str); return nil }
func (x *_error) Interface() (v interface{})         { return x.e }
func (x *_error) Error() string                      { return x.e.Error() }
func (x _errorWrap) Unserialize(string) error        { return nil }
func (x _errorWrap) String() string                  { return "" }

type (
	_float64      float64
	_float64Param struct{ SerializableParam }
)

func Float64(v float64) *_float64                      { x := _float64(v); return &x }
func (x *_float64) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_float64) Serialize() (str string, err error) { return stringify(x), nil }
func (x *_float64) Unserialize(str string) (err error) {
	v, err := strconv.ParseFloat(str, 64)
	*x = _float64(v)
	return err
}
func (x *_float64) Interface() (v interface{})   { return float64(*x) }
func (x _float64Param) Unserialize(string) error { return nil }
func (x _float64Param) String() string           { return "" }
func (x _float64) Param() Serializable           { v := _float64(0); return &v }

type (
	_int      int
	_intParam struct{ SerializableParam }
)

func Integer(v int) *_int                          { x := _int(v); return &x }
func (x *_int) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_int) Serialize() (str string, err error) { return stringify(x), nil }
func (x *_int) Unserialize(str string) (err error) {
	v, err := strconv.ParseInt(str, 10, bits.UintSize)
	if err != nil {
		return err
	}
	*x = _int(v)
	return nil
}
func (x *_int) Interface() (v interface{})   { return int(*x) }
func (x _intParam) Unserialize(string) error { return nil }
func (x _intParam) String() string           { return "" }
func (x _int) Param() Serializable           { v := _int(0); return &v }

type (
	_map      map[string]interface{}
	_mapParam struct{ SerializableParam }
)

func Map(m map[string]interface{}) _map           { return _map(m) }
func (x _map) String() (str string)               { str, _ = x.Serialize(); return }
func (x _map) Serialize() (str string, err error) { b, err := json.Marshal(x); return string(b), err }
func (x _map) Unserialize(str string) (err error) { return json.Unmarshal([]byte(str), &x) }
func (x _map) Interface() (v interface{})         { return (map[string]interface{})(x) }
func (x _mapParam) Unserialize(string) error      { return nil }
func (x _mapParam) String() string                { return "" }
func (x _map) Param() Serializable                { return _map{} }

type (
	_string      string
	_stringParam struct{ SerializableParam }
)

func String(v string) *_string                        { x := _string(v); return &x }
func (x *_string) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_string) Serialize() (str string, err error) { return stringify(x), nil }
func (x *_string) Unserialize(str string) (err error) { *x = _string(str); return nil }
func (x *_string) Interface() (v interface{})         { return string(*x) }
func (x _stringParam) Unserialize(string) error       { return nil }
func (x _stringParam) String() string                 { return "" }
func (x _string) Param() Serializable                 { v := _string(""); return &v }

type (
	_uint      uint
	_uintParam struct{ SerializableParam }
)

func Uinteger(v uint) *_uint                        { x := _uint(v); return &x }
func (x *_uint) String() (str string)               { str, _ = x.Serialize(); return }
func (x *_uint) Serialize() (str string, err error) { return stringify(x), nil }
func (x *_uint) Unserialize(str string) (err error) {
	v, err := strconv.ParseUint(str, 10, bits.UintSize)
	if err != nil {
		return err
	}
	*x = _uint(v)
	return nil
}
func (x *_uint) Interface() (v interface{})   { return uint(*x) }
func (x _uintParam) Unserialize(string) error { return nil }
func (x _uintParam) String() string           { return "" }
func (x _uint) Param() Serializable           { v := _uint(0); return &v }

type Convert []Serializable

func (in Convert) ToInterface() []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i], _ = v.Serialize()
	}
	return out
}
