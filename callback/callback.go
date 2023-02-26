package callback

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	seri "github.com/njones/socketio/serialize"
)

type ErrorWrap func() error

func (fn ErrorWrap) Callback(data ...interface{}) error { return fn() }
func (ErrorWrap) Serialize() (string, error)            { return "", ErrUnimplementedSerialize }
func (ErrorWrap) Unserialize(string) error              { return ErrUnimplementedUnserialize }

type FuncAny func(...interface{}) error

func (fn FuncAny) Callback(v ...interface{}) error { return fn(v...) }
func (FuncAny) Serialize() (string, error)         { return "", ErrUnimplementedSerialize }
func (FuncAny) Unserialize(string) error           { return ErrUnimplementedUnserialize }

type FuncAnyAck func(...interface{}) []seri.Serializable

func (fn FuncAnyAck) Callback(v ...interface{}) error { return ErrUnimplementedSerialize }
func (fn FuncAnyAck) CallbackAck(v ...interface{}) []interface{} {
	slice := fn(v...)
	out := make([]interface{}, len(v))
	for i, ice := range slice {
		if x, ok := ice.(interface{ Interface() interface{} }); ok {
			out[i] = x.Interface()
		}
	}
	return out
}
func (FuncAnyAck) Serialize() (string, error) { return "", ErrUnimplementedSerialize }
func (FuncAnyAck) Unserialize(string) error   { return ErrUnimplementedUnserialize }

type FuncString func(string)

func (fn FuncString) Callback(v ...interface{}) error {
	if len(v) == 0 {
		v = append(v, "unknown")
	}
	if val, ok := v[0].(string); ok {
		fn(val)
	} else {
		fn("undefined")
	}
	return nil
}
func (FuncString) Serialize() (string, error) { return "", ErrUnimplementedSerialize }
func (FuncString) Unserialize(string) error   { return ErrUnimplementedUnserialize }

type Wrap struct {
	Func       func() interface{} // func([T]...) error
	Parameters []seri.Serializable
}

func (fn Wrap) Callback(data ...interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case string:
				err = errors.New(e)
			case error:
				err = e
			default:
				// Fallback err (per specs, error strings should be lowercase w/o punctuation
				err = ErrUnknownPanic
			}
		}
	}()

	f := reflect.ValueOf(fn.Func())

	if len(data) != f.Type().NumIn() {
		return ErrUnexpectedDataInParams.F(len(data), f.Type().NumIn())
	}

	if len(fn.Parameters) != f.Type().NumIn() {
		return ErrUnexpectedFuncInParams.F(len(fn.Parameters), f.Type().NumIn())
	}

	if f.Type().NumOut() != 1 {
		return ErrUnexpectedSingleOutParam.F(f.Type().NumOut())
	}

	type inter interface{ Interface() interface{} }
	type param interface{ Param() seri.Serializable }

	in := make([]reflect.Value, f.Type().NumIn())
	for i, val := range fn.Parameters {
		if mint, ok := val.(param); ok {
			val = mint.Param()
		}

		var v string
		switch data[i].(type) {
		case error:
			in[i] = reflect.ValueOf(data[i].(error))
		case io.Reader:
			in[i] = reflect.ValueOf(data[i].(io.Reader))
		default:
			v = fmt.Sprintf("%v", data[i]) // this should work for scalar types
			val.Unserialize(v)
			in[i] = reflect.ValueOf(val.(inter).Interface())
		}

	}

	res := f.Call(in)
	rtnErr := res[0].Interface()
	if rtnErr != nil {
		return rtnErr.(error)
	}

	return nil
}

func (Wrap) Serialize() (string, error) { return "", ErrUnimplementedSerialize }
func (Wrap) Unserialize(string) error   { return ErrUnimplementedUnserialize }
