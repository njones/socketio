//go:build gc || (eio_svr_v3 && eio_svr_v4 && eio_svr_v5)
// +build gc eio_svr_v3,eio_svr_v4,eio_svr_v5

package engineio

import (
	"net/http"
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

func init() { registry[Version3.Int()] = NewServerV3 }

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

func NewServerV3(opts ...Option) Server {
	v3 := (&serverV3{}).new(opts...)
	v3.With(opts...)

	return v3
}

func (v3 *serverV3) new(opts ...Option) *serverV3 {
	v3.serverV2 = (&serverV2{}).new(opts...)

	v3.pingTimeout = 5000 * time.Millisecond
	v3.pingInterval = 25000 * time.Millisecond

	v3.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV3,
		PacketDecoder:  eiop.NewPacketDecoderV3,
		PayloadEncoder: eiop.NewPayloadEncoderV3,
		PayloadDecoder: eiop.NewPayloadDecoderV3,
	}

	v3.cors.enable = true

	if v3.servers == nil {
		v3.servers = make(map[EIOVersionStr]server)
	}
	v3.servers[Version3] = v3

	return v3
}

func (v3 *serverV3) With(opts ...Option) {
	v3.serverV2.With(opts...)
	for _, opt := range opts {
		opt(v3)
	}
}

func (v3 *serverV3) serveTransport(w http.ResponseWriter, r *http.Request) (transport eiot.Transporter, err error) {
	ctx := r.Context()

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
		sessionID = v3.generateID()

		transportName := transportNameFrom(r)
		transport = v3.transports[transportName](sessionID, v3.codec)
		if err := v3.sessions.Set(transport); err != nil {
			return nil, err
		}

		transport.Send(v3.handshakePacket(sessionID, transportName))
		if v3.initialPackets != nil {
			v3.initialPackets(transport, r)
		}

		if t, ok := transport.(interface {
			Write(http.ResponseWriter, *http.Request) error
		}); ok {

			ctx = v3.sessions.WithInterval(ctx, v3.pingInterval)
			ctx = v3.sessions.WithTimeout(ctx, v3.pingTimeout)

			transport.Send(eiop.Packet{T: eiop.NoopPacket, D: eiot.WriteClose{}})
			return transport, ToEOH(t.Write(w, r.WithContext(ctx)))
		}
	}

	upgrade := v3.doUpgrade(v3.sessions.Get(sessionID))(w, r)
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

	ctx = v3.sessions.WithInterval(ctx, v3.pingInterval)
	ctx = v3.sessions.WithTimeout(ctx, v3.pingTimeout)

	go func() {
		v3.transportRunError <- upgrade.transport.Run(w, r.WithContext(ctx), append(v3.eto, opts...)...)
	}()

	return upgrade.transport, nil
}

func (v3 *serverV3) handshakePacket(sessionID SessionID, transportName TransportName) eiop.Packet {
	packet := v3.serverV2.handshakePacket(sessionID, transportName)
	packet.D = &eiop.HandshakeV3{HandshakeV2: packet.D.(*eiop.HandshakeV2), PingInterval: eiop.Duration(v3.pingInterval)}
	return packet
}
