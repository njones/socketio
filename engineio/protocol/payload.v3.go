//go:build gc || (eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v3,eio_pay_v4

package protocol

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// PayloadV3 is defined: https://github.com/socketio/engine.io-protocol/tree/v3
type PayloadV3 []PacketV3

type PayloadDecoderV3 struct {
	*PayloadDecoderV2

	IsXHR2 bool
}

var NewPayloadDecoderV3 _payloadDecoderV3 = func(r io.Reader) *PayloadDecoderV3 {
	return &PayloadDecoderV3{PayloadDecoderV2: &PayloadDecoderV2{r: r}}
}

func (dec *PayloadDecoderV3) Decode(payV3 *PayloadV3) (err error) {
	if payV3 == nil {
		payV3 = &PayloadV3{}
	}
	if dec.IsXHR2 {
		for {
			pr := dec.decode(dec.r)
			var pacV3 PacketV3
			var pacType [1]byte
			var msgType string

			pr.Read(pacType[:])
			if pacType[0] == 1 {
				pacV3.IsBinary = true
				msgType = "4" // fake and hardcode this to force a message type... this is not in the spec...
			}

			if err = NewPacketDecoderV3(io.MultiReader(strings.NewReader(msgType), pr)).Decode(&pacV3); err != nil {
				if errors.Is(err, io.EOF) {
					err = nil
				}
				return ErrPayloadDecode.F("v3", err)
			}

			*payV3 = append(*payV3, pacV3)
		}
	}

	for {
		pr := dec.PayloadDecoderV2.decode(dec.r)
		var pacV3 PacketV3
		var pacType [1]byte
		var r io.Reader

		if _, err := pr.Read(pacType[:]); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return ErrPayloadDecode.F("v3", err)
		}
		if pacType[0] == 'b' {
			pacV3.IsBinary = true
			pr.Read(pacType[:]) // read the message type
			r = io.MultiReader(bytes.NewReader(pacType[:]), base64.NewDecoder(base64.StdEncoding, pr))
		} else {
			r = io.MultiReader(bytes.NewReader(pacType[:]), pr)
		}

		if err = NewPacketDecoderV3(r).Decode(&pacV3); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return ErrPayloadDecode.F("v3", err)
		}

		*payV3 = append(*payV3, pacV3)
	}
}

func (dec *PayloadDecoderV3) decode(src io.Reader) io.Reader {
	var (
		b    [1]byte
		numB []byte
		numI uint64
	)
	for {
		if _, err := src.Read(b[:]); err != nil {
			return src
		}
		if b[0] == 0xFF {
			i := Btol(numB)
			numI = i
			break
		}
		numB = append(numB, b[0])
	}

	r := io.LimitReader(src, int64(numI))
	return r
}

type PayloadEncoderV3 struct {
	*PayloadEncoderV2

	IsXHR2 bool
}

var NewPayloadEncoderV3 _payloadEncoderV3 = func(w io.Writer) *PayloadEncoderV3 {
	return &PayloadEncoderV3{PayloadEncoderV2: &PayloadEncoderV2{w: w}}
}

func (enc *PayloadEncoderV3) Encode(payload PayloadV3) error {
	for _, packet := range payload {
		if err := enc.encode(packet); err != nil {
			return ErrPayloadEncode.F("v3", err)
		}
	}
	return nil
}

func (enc *PayloadEncoderV3) encode(packet PacketV3) error {
	if enc.IsXHR2 {
		var isBinary bool
		switch packet.T {
		case MessagePacket:
			_, isBinary = packet.D.([]byte)
		}

		var buf = new(bytes.Buffer)
		if err := NewPacketEncoderV3(buf).Encode(packet); err != nil {
			return ErrPayloadEncode.F("v3", err)
		}

		// NOTE: in the spec the binary payload doesn't include the packet type byte.
		// which is conviently the first byte, so if we start on byte 1 we skip it, which
		// is 1 and is the binary type indicator. So we use the binary indicator to know where
		// to start. That's why things are a bit funky looking here...
		header := head(uint64(buf.Len()), isBinary)
		if _, err := enc.w.Write(append(header, buf.Bytes()[header[0]:]...)); err != nil {
			return ErrPayloadEncode.F("v3", err)
		}
		return nil
	}

	switch packet.T {
	case MessagePacket: // the only place for binary data in v3
		var data = new(strings.Builder)
		switch value := packet.D.(type) {
		case []byte:
			b64 := base64.NewEncoder(base64.StdEncoding, data)
			b64.Write(value)
			b64.Close()
			if _, err := fmt.Fprintf(enc.w, "%d:b4%s", data.Len()+2, data.String()); err != nil {
				return ErrPayloadEncode.F("v3", err)
			}
			return nil
		}
	}

	return enc.PayloadEncoderV2.encode(PacketV2{Packet: packet.Packet})

}
