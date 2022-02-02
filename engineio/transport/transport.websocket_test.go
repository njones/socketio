package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	eiop "github.com/njones/socketio/engineio/protocol"
)

func TestWebsocketTransportReceive(t *testing.T) {
	c := Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}
	tr := NewWebsocketTransport()(SessionID("12345"), c)

	q := new(sync.WaitGroup)
	h := make(chan eiop.Packet, 1)

	q.Add(1)
	go func() {
		defer q.Done()
		h <- <-tr.Receive()
	}()

	wai := new(sync.WaitGroup)
	wai.Add(1)
	server := httptest.NewServer(testRunHandler{wait: wai, t: t, fn: tr.Run})

	wsURL := strings.ReplaceAll(server.URL, "http", "ws")
	wsConn, _, _, err := ws.Dial(context.TODO(), wsURL+"/engine.io")
	wai.Done()

	if err != nil {
		t.Fatal(err)
	}

	ping, err := wsutil.ReadServerText(wsConn)
	if err != nil {
		t.Fatal(err)
	}

	t.Error("PING", ping)

	err = wsutil.WriteClientText(wsConn, append([]byte{'3'}, ping[1:]...))
	if err != nil {
		t.Fatal(err)
	}

	err = wsutil.WriteClientText(wsConn, []byte("4Hello"))
	if err != nil {
		t.Fatal(err)
	}

	q.Wait()
	t.Error(<-h)
}

type testRunHandler struct {
	fn   func(http.ResponseWriter, *http.Request, ...Option) error
	t    *testing.T
	opts []Option
	wait *sync.WaitGroup
}

func (h testRunHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = context.WithValue(ctx, serverSetupComplete, h.wait)
	h.t.Error(h.fn(w, r.WithContext(ctx), h.opts...))
}
