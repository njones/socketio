//go:build gc || (eio_pay_v2 && eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v2,eio_pay_v3,eio_pay_v4

package protocol

import (
	"io"
)

func (pay PayloadV2) PayloadVal() Payload {
	rtn := make(Payload, len(pay))
	for i, v := range pay {
		rtn[i] = v.Packet
	}
	return rtn
}

type _payloadDecoderV2 func(r io.Reader) *PayloadDecoderV2
type _payloadEncoderV2 func(w io.Writer) *PayloadEncoderV2
type _payloadReaderV2 func(pay *PayloadV2) (err error)
type _payloadWriterV2 func(pay PayloadV2) (err error)

func (pay _payloadDecoderV2) From(r io.Reader) PayloadReader { return _payloadReaderV2(pay(r).Decode) }
func (pay _payloadEncoderV2) To(w io.Writer) PayloadWriter   { return _payloadWriterV2(pay(w).Encode) }
func (pay _payloadReaderV2) ReadPayload(payload PayloadRef) (err error) {
	var pay2 PayloadV2
	if err = pay(&pay2); err != nil {
		return err
	}
	payRef := make([]Packet, len(pay2))
	for i, v := range pay2 {
		payRef[i] = v.Packet
	}
	*payload.PayloadRef() = payRef
	return nil
}
func (pay _payloadWriterV2) WritePayload(payload PayloadVal) error {
	switch pay2 := payload.(type) {
	case PayloadV2:
		return pay(pay2)
	}
	pay2 := make(PayloadV2, len(payload.PayloadVal()))
	for i, v := range payload.PayloadVal() {
		pay2[i] = PacketV2{Packet: v}
	}
	return pay(pay2)
}
