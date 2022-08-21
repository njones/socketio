package engineio

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConnectionServerV3(t *testing.T) {
	v3 := NewServerV3()

	server := httptest.NewServer(v3)
	client := http.DefaultClient
	urlStr := fmt.Sprintf("%s/engine.io/?EIO=3&transport=polling&t=%d", server.URL, time.Now().UnixNano())

	resp, err := client.Get(urlStr)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

}
