//go:build gc || (eio_svr_v2 && eio_svr_v3 && eio_svr_v4 && eio_svr_v5)
// +build gc eio_svr_v2,eio_svr_v3,eio_svr_v4,eio_svr_v5

package engineio

//
// https://github.com/socketio/engine.io-protocol/tree/v2
// https://github.com/socketio/engine.io/tree/1.8.x
// https://github.com/socketio/engine.io/tree/2.1.x
// https://github.com/socketio/engine.io/compare/1.8.x...2.1.x
//

import (
	"errors"
	"net/http"
	"strings"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	sess "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
)

const Version2 EIOVersionStr = "2"

func init() { registery[Version2.Int()] = NewServerV2 }

type serverV2 struct {
	path *string

	allowUpgrades     bool
	pingTimeout       time.Duration
	upgradeTimeout    time.Duration
	maxHttpBufferSize int
	cookie            struct {
		name     string
		path     string
		httpOnly bool
	}

	transportChanBuf int

	initialPacket func() eiop.Packet
	generateID    func() SessionID

	codec eiot.Codec

	eto []eiot.Option

	servers    map[EIOVersionStr]server
	sessions   mapSessionToTransport
	transports map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter
}

func NewServerV2(opts ...Option) Server { return (&serverV2{}).new(opts...) }

func (v2 *serverV2) new(opts ...Option) *serverV2 {
	v2.path = amp("/engine.io")
	v2.allowUpgrades = true
	v2.pingTimeout = 60000 * time.Millisecond
	v2.upgradeTimeout = 10000 * time.Millisecond
	v2.maxHttpBufferSize = 10e7
	v2.transportChanBuf = 1000

	v2.eto = append(v2.eto, eiot.WithPingTimeout(v2.pingTimeout))

	v2.generateID = sess.GenerateID
	v2.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	v2.servers = make(map[EIOVersionStr]server)
	v2.sessions = NewSessionMap()
	v2.transports = make(map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter)

	WithTransport("polling", eiot.NewPollingTransport(v2.transportChanBuf, time.Second*3))(v2)
	WithTransport("websocket", eiot.NewWebsocketTransport(v2.transportChanBuf))(v2)

	v2.With(v2, opts...)

	return v2
}

func (v2 *serverV2) With(svr Server, opts ...Option) {
	for _, opt := range opts {
		opt(svr)
	}
}

func (v2 *serverV2) ServeTransport(w http.ResponseWriter, r *http.Request) (eiot.Transporter, error) {
	if v2.path == nil || !strings.HasPrefix(r.URL.Path, *v2.path) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil, ErrURIPath
	}

	sessionID := sessionIDFrom(r)
	if sessionID == "" {
		return v2.initHandshake(w, r)
	}

	transport, err := v2.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	go func() {
		transport.Run(w, r, v2.eto...) // skip this error
	}()

	return transport, nil
}

func (v2 *serverV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if v2.path == nil || !strings.HasPrefix(r.URL.Path, *v2.path) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if v2.cookie.name != "" {
		cookie := http.Cookie{
			Name:     v2.cookie.name,
			Path:     v2.cookie.path,
			HttpOnly: v2.cookie.httpOnly,
		}
		r.AddCookie(&cookie)
	}

	eioVersion := eioVersionFrom(r)
	if v, ok := v2.servers[eioVersion]; ok {
		if err := v.serveHTTP(w, r); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

func (v2 *serverV2) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	if origin := r.Header.Get("Origin"); origin != "" {
		// Automatically add a CORS header
		if strings.EqualFold(origin, r.URL.Host) {
			w.Header().Set("Access-Control-Allow-Origin", r.URL.Host)
		}
	}
	if strings.ToUpper(r.Method) == "OPTIONS" {
		return nil
	}

	sessionID := sessionIDFrom(r)

	if sessionID == "" {
		_, err := v2.initHandshake(w, r)
		if errors.Is(err, EOH) {
			return nil
		}
		return err
	}

	toTransport := transportNameFrom(r)

	transport, err := v2.sessions.Get(sessionID)
	if err != nil {
		return err
	}

	if v2.allowUpgrades {
		if tport, ok := v2.doUpgrade(transport, toTransport); ok {
			transport.Shutdown() // the previous transport should stop, now overwrite it...
			transport = tport    // no shadowing, we want to replace the transport...
		}
	}

	if err = transport.Run(w, r); err != nil {
		return ErrTransportRun.F(err)
	}

	return err
}

func (v2 *serverV2) initHandshake(w http.ResponseWriter, r *http.Request) (eiot.Transporter, error) {
	sessionID := v2.generateID()
	transportName := transportNameFrom(r)

	handshakePacket := eiop.Packet{
		T: eiop.OpenPacket,
		D: &eiop.HandshakeV2{
			SID:         sessionID.String(),
			Upgrades:    v2.upgradeable(transportName, v2.transports),
			PingTimeout: eiop.Duration(v2.pingTimeout),
		},
	}

	packets := []eiop.Packet{handshakePacket}
	if v2.initialPacket != nil {
		packets = append(packets, v2.initialPacket())
	}

	transportFunc, ok := v2.transports[transportName]
	if !ok {
		return nil, ErrNoTransport
	}

	transport := transportFunc(sessionID, v2.codec)
	v2.sessions.Set(transport)

	if err := v2.codec.PayloadEncoder.To(w).WritePayload(eiop.Payload(packets)); err != nil {
		return nil, ErrPayloadEncode.F(err)
	}

	// End Of Handshake
	return transport, EOH
}

func (v2 *serverV2) doUpgrade(t eiot.Transporter /*id SessionID, from,*/, to eiot.Name) (eiot.Transporter, bool) {
	id := t.ID()
	from := t.Name()

	if to == from {
		return nil, false
	}

	for _, val := range v2.upgradeable(from, v2.transports) {
		if string(to) == val {
			return v2.transports[to](id, v2.codec), true
		}
	}

	return nil, false
}

func (v2 *serverV2) upgradeable(name eiot.Name, tps map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter) []string {
	if v2.allowUpgrades {
		switch name {
		case "polling":
			var rtn []string
			for key := range tps {
				if key == name {
					continue
				}
				rtn = append(rtn, string(key))
			}
			return rtn
		}
	}
	return nil
}
