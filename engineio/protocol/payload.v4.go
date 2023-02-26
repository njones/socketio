//go:build gc || eio_pay_v4
// +build gc eio_pay_v4

package protocol

import (
	"io"

	rw "github.com/njones/socketio/internal/readwriter"
)

type PayloadV4 []PacketV4

type PayloadDecoderV4 struct{ *PayloadDecoderV3 }

var NewPayloadDecoderV4 _payloadDecoderV4 = func(r io.Reader) *PayloadDecoderV4 {
	return &PayloadDecoderV4{
		PayloadDecoderV3: &PayloadDecoderV3{
			PayloadDecoderV2: &PayloadDecoderV2{read: &reader{Reader: rw.NewReader(r)}},
		},
	}
}

func (dec *PayloadDecoderV4) Decode(payload *PayloadV4) (err error) {
	if payload == nil {
		payload = &PayloadV4{}
	}

	var r = newRecordScan(dec.read)

	for dec.read.IsNotErr() {
		var packet PacketV4
		if dec.read.IsNotErr() && dec.read.ConditionalErr(NewPacketDecoderV4(r).Decode(&packet)).IsNotErr() {
			*payload = append(*payload, packet)
		}
	}

	return dec.read.ConvertErr(io.EOF, nil).Err()
}

type PayloadEncoderV4 struct{ *PayloadEncoderV3 }

var NewPayloadEncoderV4 _payloadEncoderV4 = func(w io.Writer) *PayloadEncoderV4 {
	return &PayloadEncoderV4{
		PayloadEncoderV3: &PayloadEncoderV3{
			PayloadEncoderV2: &PayloadEncoderV2{write: &writer{Writer: rw.NewWriter(w)}},
		},
	}
}

func (enc *PayloadEncoderV4) Encode(payload PayloadV4) error {
	for i, packet := range payload {
		if i > 0 {
			enc.write.Byte(RecordSeparator).OnErrF(ErrEncodePayloadFailed)
		}
		if err := enc.encode(packet); err != nil {
			return ErrEncodePayloadFailed.F(err).KV(ver, "v4")
		}
	}
	return enc.write.Err()
}

func (enc *PayloadEncoderV4) encode(packet PacketV4) error {
	if err := NewPacketEncoderV4(enc.write).Encode(packet); err != nil {
		return ErrEncodePayloadFailed.F(err).KV(ver, "v4")
	}
	return nil
}
