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
	"context"
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

type upgradeable struct {
	transport     eiot.Transporter
	isProbeOnInit bool
	upgradeFn     func() error
	err           error
}

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
	sessions   transportSessions
	transports map[TransportName]func(SessionID, eiot.Codec) eiot.Transporter

	transportRunError chan error
}

func NewServerV2(opts ...Option) Server {
	v2 := (&serverV2{}).new(opts...)
	v2.With(opts...)
	return v2
}

func (v2 *serverV2) new(opts ...Option) *serverV2 {
	v2.path = amp("/engine.io")
	v2.allowUpgrades = true
	v2.pingTimeout = 60000 * time.Millisecond
	v2.upgradeTimeout = 10000 * time.Millisecond
	v2.maxHttpBufferSize = 10e7
	v2.transportChanBuf = 1000
	v2.transportRunError = make(chan error, 1)

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
	v2.sessions = NewSessions()
	v2.transports = make(map[TransportName]func(SessionID, eiot.Codec) eiot.Transporter)

	WithTransport("polling", eiot.NewPollingTransport(v2.transportChanBuf))(v2)
	WithTransport("websocket", eiot.NewWebsocketTransport(v2.transportChanBuf))(v2)

	v2.eto = []eiot.Option{eiot.WithGovernor(1500*time.Microsecond, 500*time.Microsecond)}

	return v2
}

func (v2 *serverV2) With(opts ...Option) {
	for _, opt := range opts {
		opt(v2)
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
		case errors.Is(err, ErrInvalidRequestHTTPMethod):
			return
		}
		return
	}
}

func (v2 *serverV2) ServeTransport(w http.ResponseWriter, r *http.Request) (eiot.Transporter, error) {
	if v2.path == nil || !strings.HasPrefix(r.URL.Path, *v2.path) {
		return nil, ErrInvalidURIPath
	}

	sessionID := sessionIDFrom(r)

	switch r.Method {
	case http.MethodGet, http.MethodOptions:
		break
	case http.MethodPost:
		if sessionID != "" {
			break
		}
		fallthrough
	default:
		return nil, ErrInvalidRequestHTTPMethod
	}

	eioVersion := eioVersionFrom(r)
	server, ok := v2.servers[eioVersion]
	if !ok || eioVersion == "" {
		return nil, ErrUnknownEIOVersion
	}

	transportName := transportNameFrom(r)
	if _, ok := v2.transports[transportName]; !ok || transportName == "" {
		return nil, ErrUnknownTransport
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxSessionID, sessionID)
	ctx = context.WithValue(ctx, ctxTransportName, transportName)
	ctx = context.WithValue(ctx, ctxEIOVersion, eioVersion)

	transport, err := server.serveTransport(w, r.WithContext(ctx))
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
	ctx := r.Context()

	if origin := r.Header.Get("Origin"); origin != "" {
		if strings.EqualFold(origin, r.URL.Hostname()) {
			w.Header().Set("Access-Control-Allow-Origin", r.URL.Host)
		}
	}
	if strings.ToUpper(r.Method) == "OPTIONS" {
		return nil, IOR
	}

	sessionID, _ := ctx.Value(ctxSessionID).(SessionID)
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

			ctx = v2.sessions.WithTimeout(ctx, v2.pingTimeout)
			ctx = v2.sessions.WithInterval(ctx, v2.pingTimeout-10*time.Millisecond)

			return transport, ToEOH(t.Write(w, r.WithContext(ctx)))
		}
	}

	upgrade := v2.doUpgrade(v2.sessions.Get(sessionID))(w, r)
	if upgrade.err != nil {
		return nil, upgrade.err
	}

	var opts []eiot.Option
	if upgrade.isProbeOnInit {
		opts = []eiot.Option{eiot.OnInitProbe(upgrade.isProbeOnInit)}
	}
	if upgrade.upgradeFn != nil {
		opts = []eiot.Option{eiot.OnUpgrade(upgrade.upgradeFn)}
	}

	ctx = v2.sessions.WithTimeout(ctx, v2.pingTimeout*4)
	ctx = v2.sessions.WithInterval(ctx, v2.pingTimeout)

	opts = append(opts, eiot.WithNoPing())
	go func() {
		v2.transportRunError <- upgrade.transport.Run(w, r.WithContext(ctx), append(v2.eto, opts...)...)
	}()

	return upgrade.transport, nil
}

func (v2 *serverV2) handshakePacket(sessionID SessionID, transportName TransportName) eiop.Packet {
	return eiop.Packet{
		T: eiop.OpenPacket,
		D: &eiop.HandshakeV2{
			SID:         sessionID.String(),
			Upgrades:    v2.upgrades(transportName, v2.transports),
			PingTimeout: eiop.Duration(v2.pingTimeout),
		},
	}
}

func (v2 *serverV2) doUpgrade(transport eiot.Transporter, err error) func(http.ResponseWriter, *http.Request) upgradeable {
	return func(w http.ResponseWriter, r *http.Request) upgradeable {
		if err != nil {
			return upgradeable{transport: transport, err: err}
		}
		sessionID, from, to := transport.ID(), transport.Name(), transportNameFrom(r)
		if to != from {
			for _, val := range v2.upgrades(from, v2.transports) {
				if string(to) == val {
					return upgradeable{
						transport:     v2.transports[to](sessionID, v2.codec),
						isProbeOnInit: true,
						upgradeFn:     func() error { return v2.sessions.Set(transport) },
						err:           nil,
					}
				}
			}
			return upgradeable{err: ErrTransportUpgradeFailed}
		}
		return upgradeable{transport: transport, err: err}
	}
}

func (v2 *serverV2) upgrades(name TransportName, tps map[TransportName]func(SessionID, eiot.Codec) eiot.Transporter) []string {
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
