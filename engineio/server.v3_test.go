package engineio_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	eio "github.com/njones/socketio/engineio"
	eios "github.com/njones/socketio/engineio/session"
	"github.com/stretchr/testify/assert"
)

func TestServerV3(t *testing.T) {
	var opts = []func(*testing.T){}
	var EIOv = 3

	runWithOptions := map[string]testParamsInFn{
		"Service": func(v3 eio.Server, out []variables) testFn {
			return Service(opts, EIOv, v3, out)
		},
	}

	tests := map[string]testParamsOutFn{
		"basic": BasicV3,
		"cors":  CORSV3,
	}

	for name, testParams := range tests {
		for suffix, run := range runWithOptions {
			t.Run(fmt.Sprintf("%s.%s", name, suffix), run(testParams(t)))
		}
	}

}

func Service(opts []func(*testing.T), EIOv int, v3 eio.Server, out []variables) testFn {
	return func(t *testing.T) {
		for _, opt := range opts {
			opt(t)
		}

		var (
			server = httptest.NewServer(v3)
			client = server.Client()
		)

		defer server.Close()

		for _, v := range out {
			req, err := http.NewRequest(v.method, fmt.Sprintf(v.url, server.URL, EIOv), v.body)
			assert.NoError(t, err)

			for k, v := range v.headers {
				req.Header.Add(k, v)
			}

			resp, err := client.Do(req)
			assert.NoError(t, err)

			v.resp(t, resp)
		}
	}
}

func BasicV3(t *testing.T) (a eio.Server, m []variables) {
	v2 := eio.NewServerV3(
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

				assert.Equal(t, `88:0{"sid":"Apple","upgrades":["hope","websocket"],"pingTimeout":5000,"pingInterval":25000}`, buf.String())
			},
		},
	}
	return v2, out
}

func CORSV3(t *testing.T) (a eio.Server, m []variables) {
	v2 := eio.NewServerV3(
		eio.WithCors(eio.CORSorigin{"*"}),
		eio.WithGenerateIDFunc(func() eios.ID { return eios.ID("Apple") }),
	)

	out := []variables{
		{
			method:  "GET",
			url:     "%s/engine.io/?EIO=%d&transport=polling",
			headers: map[string]string{"origin": "localhost"},
			resp: func(t *testing.T, resp *http.Response) {

				// TODO(njones): fix the access control headers...

				// haveHeader := resp.Header.Get("access-control-allow-origin")
				// assert.Equal(t, "*", haveHeader)

				var buf = new(bytes.Buffer)
				n, err := buf.ReadFrom(resp.Body)
				assert.Greater(t, n, int64(0))
				assert.NoError(t, err)

				assert.Equal(t, `81:0{"sid":"Apple","upgrades":["websocket"],"pingTimeout":5000,"pingInterval":25000}`, buf.String())
			},
		},
	}
	return v2, out
}
