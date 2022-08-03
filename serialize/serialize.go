package serialize

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

// https://github.com/socketio/socket.io/tree/master/examples/custom-parsers

type Serializable interface {
	Serialize() (string, error)
	Unserialize(string) error
}

type _internal_ interface {
	Serializable
	_internalOnly()
}

type _string string
type _internal_string struct{ *_string }

func (_internal_string) _internalOnly() {}

func (x *_string) Interface() interface{}       { return string(*x) }
func (x *_string) Serialize() (string, error)   { return string(*x), nil }
func (x *_string) Unserialize(str string) error { *x = _string(str); return nil }

func String(str string) *_string { x := _string(str); return &x }

var Str _internal_ = &_internal_string{String("")}

type _int int
type _internal_int struct{ *_int }

func (_internal_int) _internalOnly() {}

func (x *_int) Interface() interface{}     { return int(*x) }
func (x *_int) Serialize() (string, error) { return fmt.Sprintf("%d", x), nil }
func (x *_int) Unserialize(str string) error {
	i, err := strconv.ParseInt(str, 10, 64)
	*x = _int(i)
	return err
}

func Int(i int) *_int { x := _int(i); return &x }

var ISize _internal_ = &_internal_int{Int(0)}

type _uint uint
type _internal_uint struct{ *_uint }

func (_internal_uint) _internalOnly() {}

func (x *_uint) Interface() interface{}     { return int(*x) }
func (x *_uint) Serialize() (string, error) { return fmt.Sprintf("%d", x), nil }
func (x *_uint) Unserialize(str string) error {
	i, err := strconv.ParseUint(str, 10, 64)
	*x = _uint(i)
	return err
}

func Uint(i int) *_uint { x := _uint(i); return &x }

var USize _internal_ = &_internal_uint{Uint(0)}

type _float float64
type _internal_float struct{ *_float }

func (_internal_float) _internalOnly() {}

func (x *_float) Interface() interface{}     { return int(*x) }
func (x *_float) Serialize() (string, error) { return fmt.Sprintf("%d", x), nil }
func (x *_float) Unserialize(str string) error {
	i, err := strconv.ParseFloat(str, 64)
	*x = _float(i)
	return err
}

func Float(f float64) *_float { x := _float(f); return &x }

var Float64 _internal_ = &_internal_float{Float(0)}

type _err struct{ error }
type _internal_err struct{ *_err }

func (_internal_err) _internalOnly() {}

func (x *_err) Interface() interface{}     { return x.error }
func (x *_err) Serialize() (string, error) { return fmt.Sprintf("%s", x.error), nil }
func (x *_err) Unserialize(str string) error {
	x.error = errors.New(str)
	return nil
}

func Error(x error) *_err { return &_err{x} }

var Err _internal_ = &_internal_err{nil}

type _binary struct{ io.Reader }
type _internal_binary struct{ *_binary }

func (_internal_binary) _internalOnly() {}

func (x *_binary) Interface() interface{} { return x.Reader }
func (x *_binary) Serialize() (string, error) {
	return "", ErrSerializeBinary.F("use the method (obj).Read(p []byte) (int, error)")
}
func (x *_binary) Unserialize(str string) error {
	return ErrUnserializeBinary.F("use the method (obj).Read(p []byte) (int, error)")
}

func Binary(x io.Reader) *_binary { return &_binary{x} }

var Bin _internal_ = &_internal_binary{nil}

type Convert []Serializable

func (in Convert) ToInterface() []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i], _ = v.Serialize()
	}
	return out
}
