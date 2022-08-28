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
	"sort"
	"strings"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	eios "github.com/njones/socketio/engineio/session"
	eiot "github.com/njones/socketio/engineio/transport"
)

const Version2 EIOVersionStr = "2"

func init() { registry[Version2.Int()] = NewServerV2 }

type serverV2 struct {
	path *string

	allowUpgrades     bool
	pingTimeout       time.Duration
	upgradeTimeout    time.Duration
	maxHttpBufferSize int

	// https://socket.io/how-to/deal-with-cookies
	cookie struct {
		name     string
		path     string
		httpOnly bool
		sameSite http.SameSite
	}

	transportChanBuf int

	initialPackets func(eiot.Transporter, *http.Request)
	generateID     func() SessionID

	codec eiot.Codec

	eto []eiot.Option

	servers    map[EIOVersionStr]server
	sessions   mapSessionToTransport
	transports map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter

	transportRunError chan error
}

func NewServerV2(opts ...Option) Server { return (&serverV2{}).new(opts...) }

func (v2 *serverV2) new(opts ...Option) *serverV2 {
	v2.path = amp("/engine.io")
	v2.allowUpgrades = true
	v2.pingTimeout = 60000 * time.Millisecond
	v2.upgradeTimeout = 10000 * time.Millisecond
	v2.maxHttpBufferSize = 10e7
	v2.transportChanBuf = 1000
	v2.transportRunError = make(chan error, 1)

	v2.eto = append(v2.eto, eiot.WithPingTimeout(v2.pingTimeout))

	v2.generateID = eios.GenerateID
	v2.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV2,
		PacketDecoder:  eiop.NewPacketDecoderV2,
		PayloadEncoder: eiop.NewPayloadEncoderV2,
		PayloadDecoder: eiop.NewPayloadDecoderV2,
	}

	if v2.servers == nil {
		v2.servers = make(map[EIOVersionStr]server)
	}
	v2.servers[Version2] = v2
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

func (v2 *serverV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := v2.ServeTransport(w, r)
	if err != nil {
		goto HandleError
	}
	err = <-v2.transportRunError

HandleError:
	if err != nil {
		switch {
		case errors.Is(err, ErrBadRequestMethod):
			return
		}
		return
	}
}

func (v2 *serverV2) ServeTransport(w http.ResponseWriter, r *http.Request) (eiot.Transporter, error) {
	if v2.path == nil || !strings.HasPrefix(r.URL.Path, *v2.path) {
		return nil, ErrURIPath
	}

	switch r.Method {
	case http.MethodGet, http.MethodOptions:
		break
	case http.MethodPost:
		if sessionIDFrom(r) != "" {
			break
		}
		fallthrough
	default:
		return nil, ErrBadRequestMethod
	}

	eioVersion := eioVersionFrom(r)
	server, ok := v2.servers[eioVersion]
	if !ok || eioVersion == "" {
		return nil, ErrNoEIOVersion
	}

	transportName := transportNameFrom(r)
	if _, ok := v2.transports[transportName]; !ok || transportName == "" {
		return nil, ErrNoTransport
	}

	transport, err := server.serveTransport(w, r)
	if err != nil {
		return nil, err
	}

	return transport, err
}

func ToEOH(err error) error {
	if err == nil {
		return EOH
	}
	return err
}

func (v2 *serverV2) serveTransport(w http.ResponseWriter, r *http.Request) (transport eiot.Transporter, err error) {
	if origin := r.Header.Get("Origin"); origin != "" {
		if strings.EqualFold(origin, r.URL.Hostname()) {
			w.Header().Set("Access-Control-Allow-Origin", r.URL.Host)
		}
	}
	if strings.ToUpper(r.Method) == "OPTIONS" {
		return nil, IOR
	}

	sessionID := sessionIDFrom(r)
	if sessionID == "" {
		sessionID = v2.generateID()
		transportName := transportNameFrom(r)
		transport = v2.transports[transportName](sessionID, v2.codec)
		if err := v2.sessions.Set(transport); err != nil {
			return nil, err
		}

		transport.Send(v2.handshakePacket(sessionID, transportName))
		if v2.initialPackets != nil {
			v2.initialPackets(transport, r)
		}

		if t, ok := transport.(interface {
			Write(http.ResponseWriter, *http.Request) error
		}); ok {
			transport.Send(eiop.Packet{T: eiop.NoopPacket, D: eiot.WriteClose{}})
			return transport, ToEOH(t.Write(w, r))
		}
	}

	transport, _, err = v2.doUpgrade(v2.sessions.Get(sessionID))(w, r)
	if err != nil {
		return nil, err
	}

	go func() { v2.transportRunError <- transport.Run(w, r, v2.eto...) }()

	return
}

func (v2 *serverV2) handshakePacket(sessionID SessionID, transportName eiot.Name) eiop.Packet {
	return eiop.Packet{
		T: eiop.OpenPacket,
		D: &eiop.HandshakeV2{
			SID:         sessionID.String(),
			Upgrades:    v2.upgrades(transportName, v2.transports),
			PingTimeout: eiop.Duration(v2.pingTimeout),
		},
	}
}

func (v2 *serverV2) doUpgrade(transport eiot.Transporter, err error) func(http.ResponseWriter, *http.Request) (eiot.Transporter, bool, error) {
	var isUpgrade bool
	return func(w http.ResponseWriter, r *http.Request) (eiot.Transporter, bool, error) {
		if err != nil {
			return transport, isUpgrade, err
		}
		sessionID, from, to := transport.ID(), transport.Name(), transportNameFrom(r)
		if to != from {
			for _, val := range v2.upgrades(from, v2.transports) {
				if string(to) == val {
					transport = v2.transports[to](sessionID, v2.codec)
					isUpgrade = true
					return transport, isUpgrade, v2.sessions.Set(transport)
				}
			}
		}
		return transport, isUpgrade, err
	}
}

func (v2 *serverV2) upgrades(name eiot.Name, tps map[eiot.Name]func(SessionID, eiot.Codec) eiot.Transporter) []string {
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
			sort.Strings(rtn)
			return rtn
		}
	}
	return nil
}
