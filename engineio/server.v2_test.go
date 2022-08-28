package engineio_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	eio "github.com/njones/socketio/engineio"
	eios "github.com/njones/socketio/engineio/session"
	itst "github.com/njones/socketio/internal/test"
	"github.com/stretchr/testify/assert"
)

type (
	testFn          func(*testing.T)
	testParamsInFn  func(eio.Server, []variables) testFn
	testParamsOutFn func(*testing.T) (eio.Server, []variables)
)

var runTest, skipTest = itst.RunTest, itst.SkipTest //lint:ignore U1000 Ignore unused function when testing

func TestServerV2(t *testing.T) {
	var opts = []func(*testing.T){}
	var EIOv = 2

	runWithOptions := map[string]testParamsInFn{
		"Server": func(v2 eio.Server, out []variables) testFn {
			return PollingTestV2(opts, EIOv, v2, out)
		},
	}

	tests := map[string]testParamsOutFn{
		"basic": BasicV2,
		"cors":  CORSV2,
	}

	for name, testParams := range tests {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}

}

func PollingTestV2(opts []func(*testing.T), EIOv int, v2 eio.Server, out []variables) testFn {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		var (
			server = httptest.NewServer(v2)
			client = server.Client()
		)

		defer server.Close()

		for _, v := range out {
			req, err := http.NewRequest(v.method, fmt.Sprintf(v.url, server.URL, EIOv), v.body)
			assert.NoError(t, err)

			for k, v := range v.headers {
				req.Header.Add(k, fmt.Sprintf(v, server.URL))
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)

			v.resp(t, resp)
		}
	}
}

type variables struct {
	method  string
	url     string
	body    io.Reader
	headers map[string]string
	resp    func(*testing.T, *http.Response)
}

func BasicV2(t *testing.T) (a eio.Server, m []variables) {
	v2 := eio.NewServerV2(
		eio.WithTransport("hope", nil),
		eio.WithGenerateIDFunc(func() eios.ID { return eios.ID("Apple") }),
	)

	out := []variables{
		{
			method: "GET",
			url:    "%s/engine.io/?EIO=%d&transport=polling",
			resp: func(t *testing.T, resp *http.Response) {
				var buf = new(bytes.Buffer)
				n, err := buf.ReadFrom(resp.Body)
				assert.Greater(t, n, int64(0))
				assert.NoError(t, err)

				assert.Equal(t, `68:0{"sid":"Apple","upgrades":["hope","websocket"],"pingTimeout":60000}`, buf.String())
			},
		},
	}
	return v2, out
}

func CORSV2(t *testing.T) (a eio.Server, m []variables) {
	v2 := eio.NewServerV2(
		eio.WithGenerateIDFunc(func() eios.ID { return eios.ID("Apple") }),
	)

	out := []variables{
		{
			method:  "GET",
			url:     "%s/engine.io/?EIO=%d&transport=polling",
			headers: map[string]string{"origin": "%s"},
			resp: func(t *testing.T, resp *http.Response) {
				var buf = new(bytes.Buffer)
				n, err := buf.ReadFrom(resp.Body)
				assert.Greater(t, n, int64(0))
				assert.NoError(t, err)

				assert.Equal(t, `61:0{"sid":"Apple","upgrades":["websocket"],"pingTimeout":60000}`, buf.String())
			},
		},
	}
	return v2, out
}
