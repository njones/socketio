package socketio_test

import (
	"testing"
)

type callbackTest1 func(string, string) error

func (cb callbackTest1) Callback(v ...interface{}) error {
	return cb(v[0].(string), v[1].(string))
}

// This is an itegration test of the whole system. The latest and greatest.

// TestNewServer tests a new server that doesn't have anything but the default
// options. This is to make sure that we have good valid defaults.
func TestNewServer(t *testing.T) {

	// svr := socketio.NewServer()

	// var cb callbackTest1 = func(a string, b string) error {
	// 	return nil
	// }

	// svr.On("blank", cb)

	// hsvr := httptest.NewServer(svr)

	// want, have := "", ""
	// assert.Equal(t, want, have)
}
