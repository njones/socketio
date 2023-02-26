//go:build gc || (eio_pay_v2 && eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v2,eio_pay_v3,eio_pay_v4

package protocol

import (
	"io"
	"strconv"

	rw "github.com/njones/socketio/internal/readwriter"
)

// PayloadV2 is defined: https://github.com/socketio/engine.io-protocol/tree/v2
type PayloadV2 []PacketV2

type PayloadDecoderV2 struct{ read *reader }

var NewPayloadDecoderV2 _payloadDecoderV2 = func(r io.Reader) *PayloadDecoderV2 {
	return &PayloadDecoderV2{read: &reader{Reader: rw.NewReader(r)}}
}

func (dec *PayloadDecoderV2) Decode(payload *PayloadV2) error {
	if payload == nil {
		payload = &PayloadV2{}
	}

	for dec.read.IsNotErr() {

		n := dec.read.packetLen()

		var packet PacketV2
		if dec.read.IsNotErr() && dec.read.ConditionalErr(NewPacketDecoderV2(dec.read.payload(n)).Decode(&packet)).IsNotErr() {
			*payload = append(*payload, packet)
		}
	}

	return dec.read.ConvertErr(io.EOF, nil).Err()
}

type PayloadEncoderV2 struct{ write *writer }

var NewPayloadEncoderV2 _payloadEncoderV2 = func(w io.Writer) *PayloadEncoderV2 {
	return &PayloadEncoderV2{write: &writer{Writer: rw.NewWriter(w)}}
}

func (enc *PayloadEncoderV2) Encode(payload PayloadV2) error {
	for _, packet := range payload {
		if err := enc.encode(packet); err != nil {
			return err
		}
	}
	return nil
}

func (enc *PayloadEncoderV2) encode(packet PacketV2) (err error) {

	if err = enc.writePacketLen(packet.Packet); err != nil {
		return err
	}
	return enc.writePacket(packet)
}

func (enc *PayloadEncoderV2) writePacketLen(packet Packet) (err error) {
	messageTypeLen := len(packet.T.Bytes())
	switch data := packet.D.(type) {
	case string:
		enc.write.Bytes([]byte(strconv.Itoa(len([]rune(data))+messageTypeLen))).OnErrF(ErrEncodePayloadFailed, enc.write.Err(), kv(ver, "v2"))
	case []byte:
		enc.write.Bytes([]byte(strconv.Itoa(len(data)+messageTypeLen))).OnErrF(ErrEncodePayloadFailed, enc.write.Err(), kv(ver, "v2"))
	case useLen:
		enc.write.Bytes([]byte(strconv.Itoa(data.Len()+messageTypeLen))).OnErrF(ErrEncodePayloadFailed, enc.write.Err(), kv(ver, "v2"))
	default:
		enc.write.Bytes([]byte(strconv.Itoa(messageTypeLen))).OnErrF(ErrEncodePayloadFailed, enc.write.Err(), kv(ver, "v2"))
	}
	enc.write.Byte(':').OnErrF(ErrEncodePayloadFailed, kv(ver, "v2"))

	return enc.write.Err()
}

func (enc *PayloadEncoderV2) writePacket(packet PacketV2) (err error) {
	if err := NewPacketEncoderV2(enc.write).Encode(packet); err != nil {
		return ErrEncodePayloadFailed.F(err).KV(ver, "v2")
	}
	return enc.write.Err()
}
