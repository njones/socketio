//go:build gc || eio_pay_v4
// +build gc eio_pay_v4

package protocol

import "io"

type _payloadDecoderV4 func(r io.Reader) *PayloadDecoderV4
type _payloadEncoderV4 func(w io.Writer) *PayloadEncoderV4
type _payloadReaderV4 func(pay *PayloadV4) (err error)
type _payloadWriterV4 func(pay PayloadV4) (err error)

func (pay _payloadDecoderV4) From(r io.Reader) PayloadReader { return _payloadReaderV4(pay(r).Decode) }
func (pay _payloadEncoderV4) To(w io.Writer) PayloadWriter   { return _payloadWriterV4(pay(w).Encode) }
func (pay _payloadReaderV4) ReadPayload(payload PayloadRef) error {
	var pay4 PayloadV4
	if err := pay(&pay4); err != nil {
		return err
	}
	payRef := make([]Packet, len(pay4))
	for i, v := range pay4 {
		payRef[i] = v.Packet
	}
	*payload.PayloadRef() = payRef
	return nil
}
func (pay _payloadWriterV4) WritePayload(payload PayloadVal) error {
	pay4 := make(PayloadV4, len(payload.PayloadVal()))
	for i, v := range payload.PayloadVal() {
		pay4[i] = PacketV4{Packet: v}
	}
	return pay(pay4)
}
