package protocol

import "io"

type _payloadDecoderV3 func(r io.Reader) *PayloadDecoderV3
type _payloadEncoderV3 func(w io.Writer) *PayloadEncoderV3
type _payloadReaderV3 func(pay *PayloadV3) (err error)
type _payloadWriterV3 func(pay PayloadV3) (err error)

func (pay _payloadDecoderV3) From(r io.Reader) PayloadReader { return _payloadReaderV3(pay(r).Decode) }
func (pay _payloadEncoderV3) To(w io.Writer) PayloadWriter   { return _payloadWriterV3(pay(w).Encode) }
func (pay _payloadReaderV3) ReadPayload(payload PayloadRef) error {
	var pay3 PayloadV3
	if err := pay(&pay3); err != nil {
		return err
	}
	payRef := make([]Packet, len(pay3))
	for i, v := range pay3 {
		payRef[i] = v.Packet
	}
	*payload.PayloadRef() = payRef
	return nil
}
func (pay _payloadWriterV3) WritePayload(payload PayloadVal) error {
	pay3 := make(PayloadV3, len(payload.PayloadVal()))
	for i, v := range payload.PayloadVal() {
		pay3[i] = PacketV3{Packet: v}
	}
	return pay(pay3)
}
