package socketio_test

// THIS FILE DOES NOT CONTAIN TESTS...
// this file contains utilities for tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/njones/socketio"
	eio "github.com/njones/socketio/engineio"
	eiot "github.com/njones/socketio/engineio/transport"
	"github.com/stretchr/testify/assert"
)

type (
	testFn          func(*testing.T)
	testParamsInFn  func(socketio.Server, int, map[string][][]string, *sync.WaitGroup) testFn
	testParamsOutFn func(*testing.T) (socketio.Server, int, map[string][][]string, *sync.WaitGroup)
)

var (
	testingName = strings.NewReplacer(" ", "_")
	_, _        = runTest, skipTest
)

func checkCount(t *testing.T, count int) {
	if !assert.Greater(t, count, 0, "%s: make sure that the want map key is correct", t.Name()) {
		t.SkipNow()
	}
}

func runTest(testNames ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			if testName == "" || testName == "*" {
				return
			}

			want := testingName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				return
			}
		}
		t.SkipNow()
	}
}

func skipTest(testNames ...string) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		have := strings.SplitN(t.Name(), "/", 2)[1]
		suffix := strings.Split(have, ".")[1]

		for _, testName := range testNames {
			want := testingName.Replace(testName)
			if !strings.Contains(want, ".") {
				want += "." + suffix
			}
			if have == want {
				t.SkipNow()
			}
		}
	}
}

type testBinaryEventFunc func(io.Reader)

func (fn testBinaryEventFunc) Callback(v ...interface{}) error {
	fn(v[0].(io.Reader))
	return nil
}

var testingQuickPoll = eio.WithTransport("polling", eiot.NewPollingTransport(1000, 5*time.Millisecond))

type pollingClient interface {
	connect(...[]string)
	send(io.Reader)
	grab() []string
}

type testClient struct {
	polling pollingClient
}

type v1PollingClient struct {
	t *testing.T

	eioVersion   int
	eioSessionID string // filled in on connect

	base   string
	buffer *bytes.Buffer
	client *http.Client
}

func (c *v1PollingClient) ts() int64 { c.t.Helper(); return time.Now().UnixNano() }
func (c *v1PollingClient) parse(body []byte) (rtn [][]byte) {
	c.t.Helper()

	var x int
	for i := 0; i < len(body); i++ {
		switch b := body[i]; b {
		case ':':
			i++
			rtn = append(rtn, body[i:i+x])
			i, x = i+x-1, 0 // -1 is because of the i++ loop
		default:
			x *= 10
			y := int(b - '0')
			if y < 0 || y > 9 {
				assert.Fail(c.t, "parse: %s at idx: %d", string(body), i)
			}
			x += y
		}
	}
	return rtn
}
func (c *v1PollingClient) connect(queryStr ...[]string) {
	c.t.Helper()

	var query string
	if len(queryStr[0]) > 0 {
		query = "&" + strings.Join(queryStr[0], "&")
	}

	URL := fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&t=%d%s", c.base, c.eioVersion, c.ts(), query)
	have := c.get(URL)
	assert.NotEmpty(c.t, have)
	assert.Equal(c.t, byte('0'), have[0][0])

	var m map[string]interface{}
	err := json.Unmarshal(have[0][1:], &m)
	assert.NoError(c.t, err)

	c.eioSessionID, _ = m["sid"].(string)
	assert.NotEmpty(c.t, c.eioSessionID)
}
func (c *v1PollingClient) send(body io.Reader) {
	c.t.Helper()

	URL := fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&sid=%s&t=%d", c.base, c.eioVersion, c.eioSessionID, c.ts())
	resp, err := c.client.Post(URL, "text/plain", body)
	assert.NoError(c.t, err)

	assert.Equal(c.t, 200, resp.StatusCode)
}
func (c *v1PollingClient) grab() (rtn []string) {
	c.t.Helper()

	URL := fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&sid=%s&t=%d", c.base, c.eioVersion, c.eioSessionID, c.ts())
	have := c.get(URL)
	for i := range have {
		rtn = append(rtn, string(have[i]))
	}
	return rtn
}

func (c *v1PollingClient) get(URL string) [][]byte {
	c.t.Helper()

	resp, err := c.client.Get(URL)
	assert.NoError(c.t, err)

	assert.Equal(c.t, 200, resp.StatusCode)

	c.buffer.Reset()
	_, err = c.buffer.ReadFrom(resp.Body)
	assert.NoError(c.t, err)

	have := c.parse(c.buffer.Bytes())

	return have
}

type v3PollingClient struct {
	t *testing.T

	keep40s bool

	eioVersion   int
	eioSessionID string // filled in on connect

	base   string
	buffer *bytes.Buffer
	client *http.Client
}

func (c *v3PollingClient) ts() int64 { c.t.Helper(); return time.Now().UnixNano() }
func (c *v3PollingClient) connect(extraStr ...[]string) {
	c.t.Helper()

	var query string
	if len(extraStr[0]) > 0 {
		query = "&" + strings.Join(extraStr[0], "&")
	}
	URL := fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&t=%d%s", c.base, c.eioVersion, c.ts(), query)
	have := c.get(URL)
	assert.NotEmpty(c.t, have)
	assert.Equal(c.t, byte('0'), have[0][0])

	var m map[string]interface{}
	err := json.Unmarshal(have[0][1:], &m)
	assert.NoError(c.t, err)

	c.eioSessionID, _ = m["sid"].(string)
	assert.NotEmpty(c.t, c.eioSessionID)

	var nsConnect = "40"
	if len(extraStr[1]) > 0 {
		nsConnect = extraStr[1][0]
	}

	URL = fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&sid=%s&t=%d%s", c.base, c.eioVersion, c.eioSessionID, c.ts(), query)
	resp, err := c.client.Post(URL, "text/plain", strings.NewReader(nsConnect))
	assert.NoError(c.t, err)

	assert.Equal(c.t, 200, resp.StatusCode)
}
func (c *v3PollingClient) send(body io.Reader) {
	c.t.Helper()

	URL := fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&sid=%s&t=%d", c.base, c.eioVersion, c.eioSessionID, c.ts())
	resp, err := c.client.Post(URL, "text/plain", body)
	assert.NoError(c.t, err)

	assert.Equal(c.t, 200, resp.StatusCode)
}
func (c *v3PollingClient) grab() (rtn []string) {
	c.t.Helper()

	URL := fmt.Sprintf("%s/socket.io/?EIO=%d&transport=polling&sid=%s&t=%d", c.base, c.eioVersion, c.eioSessionID, c.ts())
	have := c.get(URL)
	for i := range have {
		if i == 0 && len(have[i]) > 1 && have[i][1] == '0' && !c.keep40s {
			continue
		}
		if len(have[i]) > 0 {
			rtn = append(rtn, string(have[i]))
		}
	}
	return rtn
}
func (c *v3PollingClient) get(URL string) [][]byte {
	c.t.Helper()

	resp, err := c.client.Get(URL)
	assert.NoError(c.t, err)

	assert.Equal(c.t, 200, resp.StatusCode)

	c.buffer.Reset()
	_, err = c.buffer.ReadFrom(resp.Body)
	assert.NoError(c.t, err)

	return bytes.Split(c.buffer.Bytes(), []byte{0x1e})
}
