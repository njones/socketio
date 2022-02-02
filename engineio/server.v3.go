package engineio

import (
	"errors"
	"net/http"
	"strings"
	"time"

	eiop "github.com/njones/socketio/engineio/protocol"
	eiot "github.com/njones/socketio/engineio/transport"
)

const Version3 EIOVersionStr = "3"

func init() { registery[Version3.Int()] = NewServerV3 }

type serverV3 struct {
	*serverV2

	pingInterval   time.Duration
	upgradeTimeout time.Duration
	// cookie struct { // configuration of the cookie that contains the client sid to send as part of handshake response headers.
	//                 // This cookie might be used for sticky-session. Defaults to not sending any cookie (false).
	//  domain   string
	//  encode   func(string) string // Specifies a function that will be used to encode a cookie's value. Since value of a cookie has a limited character set (and must be a simple string), this function can be used to encode a value into a string suited for a cookie's value.
	//  expires  time.Time  // Specifies the Date object to be the value for the Expires Set-Cookie attribute. By default, no expiration is set
	//  httpOnly bool
	//  maxAge   int
	//  path     string
	//  sameSite string
	//  secure   bool
	// }
	cors struct { // the options that will be forwarded to the cors module. Defaults to no CORS allowed.
		enable               bool
		origin               []string
		methods              []string
		headersAllow         []string
		headersExpose        []string
		credentials          bool
		maxAge               int
		preflightContinue    bool
		optionsSuccessStatus int
	}
}

func NewServerV3(opts ...Option) Server { return (&serverV3{}).new(opts...) }

func (v3 *serverV3) new(opts ...Option) *serverV3 {
	v3.serverV2 = (&serverV2{}).new(opts...)

	v3.codec = eiot.Codec{
		PacketEncoder:  eiop.NewPacketEncoderV3,
		PacketDecoder:  eiop.NewPacketDecoderV3,
		PayloadEncoder: eiop.NewPayloadEncoderV3,
		PayloadDecoder: eiop.NewPayloadDecoderV3,
	}

	v3.With(v3, opts...)
	return v3
}

func (v3 *serverV3) serveHTTP(w http.ResponseWriter, r *http.Request) error {
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
		return ErrOnTransportRun.F(err)
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
	if xPackets, ok := r.Context().Value(ckHandshakePackets).([]eiop.Packet); ok {
		packets = append(packets, xPackets...)
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
		return nil, ErrBadPath
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
