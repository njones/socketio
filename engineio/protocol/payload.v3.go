//go:build gc || (eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v3,eio_pay_v4

package protocol

import (
	"encoding/base64"
	"fmt"
	"io"

	rw "github.com/njones/socketio/internal/readwriter"
)

// PayloadV3 is defined: https://github.com/socketio/engine.io-protocol/tree/v3
type PayloadV3 []PacketV3

type PayloadDecoderV3 struct {
	*PayloadDecoderV2

	IsXHR2 bool
}

var NewPayloadDecoderV3 _payloadDecoderV3 = func(r io.Reader) *PayloadDecoderV3 {
	return &PayloadDecoderV3{PayloadDecoderV2: &PayloadDecoderV2{read: &reader{Reader: rw.NewReader(r)}}}
}

func (dec *PayloadDecoderV3) Decode(payload *PayloadV3) error {
	if payload == nil {
		payload = &PayloadV3{}
	}

	if dec.IsXHR2 {
		first := dec.read.Peek(1)
		if first[0] == 0x00 || first[0] == 0x01 {
			return dec.read.decodeXHR2(payload)
		}
	}

	for dec.read.IsNotErr() {

		n := dec.read.packetLen()
		r := dec.read.payload(n)

		var isBinary bool
		b := dec.read.Peek(1)
		if dec.read.IsNotErr() && b[0] == 'b' {
			isBinary = true
			dec.read.CopyN(io.Discard, 1).OnErr(ErrPayloadDecode) // consume and throw away the 'b' byte
			r = io.MultiReader(io.LimitReader(r, packetTypeLength), base64.NewDecoder(base64.StdEncoding, r))
		}

		var packet PacketV3
		packet.IsBinary = isBinary
		if dec.read.IsNotErr() && dec.read.ConditionalErr(NewPacketDecoderV3(r).Decode(&packet)).IsNotErr() {
			*payload = append(*payload, packet)
		}
	}

	return dec.read.ConvertErr(io.EOF, nil).Err()
}

type PayloadEncoderV3 struct {
	*PayloadEncoderV2

	IsXHR2 bool
}

var NewPayloadEncoderV3 _payloadEncoderV3 = func(w io.Writer) *PayloadEncoderV3 {
	return &PayloadEncoderV3{PayloadEncoderV2: &PayloadEncoderV2{write: rw.NewWriter(w)}}
}

func (enc *PayloadEncoderV3) Encode(payload PayloadV3) error {
	for _, packet := range payload {
		if err := enc.encode(packet); err != nil {
			return ErrPayloadEncode.F("v3", err)
		}
	}
	return enc.write.Err()
}

func (enc *PayloadEncoderV3) encode(packet PacketV3) error {
	enc.write.String(fmt.Sprintf("%d:", packet.Len()))
	if err := NewPacketEncoderV3(enc.write).Encode(packet); err != nil {
		return ErrPayloadEncode.F("v3", err)
	}
	return nil
}
