//go:build gc || (eio_svr_v3 && eio_svr_v4 && eio_svr_v5)
// +build gc eio_svr_v3,eio_svr_v4,eio_svr_v5

package engineio

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
)

// https://github.com/socketio/engine.io/tree/3.1.x
// https://github.com/socketio/engine.io/tree/3.4.x
// https://github.com/socketio/engine.io/tree/3.5.x
// https://github.com/socketio/engine.io/compare/2.1.x...3.4.x

const Version3 EIOVersionStr = "3"

func init() { registery[Version3.Int()] = NewServerV3 }

type serverV3 struct {
	*serverV2

	pingInterval time.Duration
	cors         struct { // the options that will be forwarded to the cors module. Defaults to no CORS allowed.
		enable               bool
		origin               []string
		methods              []string
		headersAllow         []string
		headersExpose        []string
		credentials          bool
		maxAge               int
		optionsSuccessStatus int
	}
}

func NewServerV3(opts ...Option) Server { return (&serverV3{}).new(opts...) }

func (v3 *serverV3) new(opts ...Option) *serverV3 {
	v3.serverV2 = (&serverV2{}).new(opts...)

	v3.pingTimeout = 5000 * time.Millisecond
	v3.pingInterval = 25000 * time.Millisecond

	v3.eto = append(v3.eto, eiot.WithPingInterval(v3.pingInterval))

	v3.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV3,
		PacketDecoder:  eiop.NewPacketDecoderV3,
		PayloadEncoder: eiop.NewPayloadEncoderV3,
		PayloadDecoder: eiop.NewPayloadDecoderV3,
	}

	v3.With(v3, opts...)
	return v3
}

func (v3 *serverV3) prev() Server { return v3.serverV2 }

func (v3 *serverV3) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	if origin := r.Header.Get("Origin"); origin != "" && v3.cors.enable {
		if v3.cors.credentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		for _, origin := range v3.cors.origin {
			// match the incoming domain, as per the request
			if origin == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				break
			}
			if strings.EqualFold(origin, r.URL.Host) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
		if len(v3.cors.methods) > 0 {
			methods := strings.ToUpper(strings.Join(v3.cors.methods, ", "))
			w.Header().Set("Access-Control-Allow-Methods", methods)
		}
		if len(v3.cors.headersAllow) > 0 {
			headersAllow := strings.Join(v3.cors.headersAllow, ", ")
			w.Header().Set("Access-Control-Allow-Headers", headersAllow)
		}
		if len(v3.cors.headersExpose) > 0 {
			headersExpose := strings.Join(v3.cors.headersExpose, ", ")
			w.Header().Set("Access-Control-Expose-Headers", headersExpose)
		}
		if v3.cors.maxAge > 0 {
			w.Header().Set("Access-Control-Expose-Headers", strconv.Itoa(v3.cors.maxAge))
		}
		if v3.cors.optionsSuccessStatus > 0 {
			w.WriteHeader(v3.cors.optionsSuccessStatus)
		}
	}
	if strings.ToUpper(r.Method) == "OPTIONS" {
		return nil
	}

	sessionID := sessionIDFrom(r)

	if sessionID == "" {
		_, err := v3.initHandshake(w, r)
		if errors.Is(err, EOH) {
			return nil
		}
		return err
	}

	toTransport := transportNameFrom(r)

	transport, err := v3.sessions.Get(sessionID)
	if err != nil {
		return err
	}

	if v3.allowUpgrades {
		if tport, ok := v3.doUpgrade(transport, toTransport); ok {
			transport.Shutdown() // the previous transport should stop, now overwrite it...
			transport = tport    // no shadowing, we want to replace the transport...
		}
	}

	if err = transport.Run(w, r); err != nil {
		return ErrTransportRun.F(err)
	}

	return err
}

func (v3 *serverV3) initHandshake(w http.ResponseWriter, r *http.Request) (eiot.Transporter, error) {
	sessionID := v3.generateID()
	transportName := transportNameFrom(r)

	handshakePacket := eiop.Packet{
		T: eiop.OpenPacket,
		D: &eiop.HandshakeV3{
			HandshakeV2: eiop.HandshakeV2{
				SID:         sessionID.String(),
				Upgrades:    v3.upgradeable(transportName, v3.transports),
				PingTimeout: eiop.Duration(v3.pingTimeout),
			},
			PingInterval: eiop.Duration(v3.pingInterval),
		},
	}

	packets := []eiop.Packet{handshakePacket}
	if v3.initialPacket != nil {
		packets = append(packets, v3.initialPacket())
	}

	transportFunc, ok := v3.transports[transportName]
	if !ok {
		return nil, ErrNoTransport
	}

	transport := transportFunc(sessionID, v3.codec)
	v3.sessions.Set(transport)

	if err := v3.codec.PayloadEncoder.To(w).WritePayload(eiop.Payload(packets)); err != nil {
		return nil, ErrPayloadEncode.F(err)
	}

	// End Of Handshake
	return transport, EOH
}

func (v3 *serverV3) ServeTransport(w http.ResponseWriter, r *http.Request) (eiot.Transporter, error) {
	if v3.path == nil || !strings.HasPrefix(r.URL.Path, *v3.path) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil, ErrURIPath
	}

	sessionID := sessionIDFrom(r)
	if sessionID == "" {
		return v3.initHandshake(w, r)
	}

	transport, err := v3.sessions.Get(sessionID)
	if err != nil {
		return nil, err
	}

	go func() { transport.Run(w, r) }()

	return transport, nil
}
