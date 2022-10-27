package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type (
	HTTPErrorString string

	String string
	Struct struct {
		e, rr error
		wrap  []error
		f     [][]interface{}
		kv    []interface{}
	}
)

func (e HTTPErrorString) As(v interface{}) bool {
	if str, ok := v.(string); ok {
		return strings.HasPrefix(string(e), str)
	}
	return false
}
func (e HTTPErrorString) Error() string               { return string(e) }
func (e HTTPErrorString) F(v ...interface{}) Struct   { return String(e).F(v...) }
func (e HTTPErrorString) KV(kv ...interface{}) Struct { return String(e).KV(kv...) }

func (e String) Error() string { return string(e) }

func (e String) F(v ...interface{}) Struct {
	var (
		f  [][]interface{}
		kv []interface{}
	)
	for i, val := range v {
		switch value := val.(type) {
		case Struct:
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

	return Struct{e: e, rr: err, f: f, kv: kv, wrap: errs}
}

func (e String) KV(kv ...interface{}) Struct {
	return Struct{e: e, rr: e, kv: kv}
}

func (e Struct) Error() string {
	str := e.rr.Error() + fmtKV(e.kv)
	for _, v := range e.f {
		if len(v) > 0 {
			str = fmt.Sprintf(str, v...)
		}
	}
	return str
}

func (e Struct) F(v ...interface{}) Struct {
	return Struct{e: e.e, rr: e.rr, kv: e.kv, f: append(e.f, v)}
}

func (e Struct) KV(kv ...interface{}) Struct {
	return Struct{e: e.e, rr: e.rr, kv: append(e.kv, kv...)}
}

func (e Struct) Is(target error) bool {
	if fn, ok := e.e.(interface{ Is(error) bool }); ok {
		if ok := fn.Is(target); ok {
			return true
		}
	}
	if eStruct, ok := target.(Struct); ok {
		return e.e.Error() == eStruct.e.Error()
	}
	if eStr, ok := target.(String); ok {
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

type Inf chan interface{}

type Group struct {
	inf Inf
	ctx context.Context
	fns []func(c Inf) error
}

func (g *Group) WithContext(ctx context.Context) { g.ctx = ctx }
func (g *Group) Add(fn func(Inf) error)          { g.fns = append(g.fns, fn) }
func (g *Group) Out() interface{}                { return <-g.inf }

func (g *Group) Err() error {
	for _, fn := range g.fns {
		select {
		case <-g.ctx.Done():
			return context.Canceled
		default:
			if err := fn(g.inf); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewGroup() *Group {
	return &Group{ctx: context.TODO(), inf: make(Inf, 1)}
}
