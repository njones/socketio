//go:build gc || eio_pay_v4
// +build gc eio_pay_v4

package protocol

import "io"

const RecordSeparator = 0x1e

func (pay PayloadV4) PayloadVal() Payload {
	rtn := make(Payload, len(pay))
	for i, v := range pay {
		if v.IsBinary {
			v.Packet.D = isBinaryPacket{D: v.Packet.D}
		}
		rtn[i] = v.Packet
	}
	return rtn
}

func (pay *PayloadV4) PayloadRef() *Payload { return &Payload{} }

type _payloadDecoderV4 func(r io.Reader) *PayloadDecoderV4
type _payloadEncoderV4 func(w io.Writer) *PayloadEncoderV4
type _payloadReaderV4 func(pay *PayloadV4) (err error)
type _payloadWriterV4 func(pay PayloadV4) (err error)

func (pay _payloadDecoderV4) From(r io.Reader) PayloadReader { return _payloadReaderV4(pay(r).Decode) }
func (pay _payloadEncoderV4) To(w io.Writer) PayloadWriter   { return _payloadWriterV4(pay(w).Encode) }
func (pay _payloadReaderV4) ReadPayload(payload PayloadRef) (err error) {
	var pay4 PayloadV4
	if err = pay(&pay4); err != nil {
		return err
	}

	switch pay := payload.(type) {
	case *Payload:
		payRef := make([]Packet, len(pay4))
		for i, v := range pay4 {
			payRef[i] = v.Packet
		}
		*pay = payRef
	case *PayloadV4:
		*pay = pay4
	}

	return nil
}
func (pay _payloadWriterV4) WritePayload(payload PayloadVal) error {
	pay4 := make(PayloadV4, len(payload.PayloadVal()))
	for i, v := range payload.PayloadVal() {
		var isBinary bool
		if binaryData, ok := v.D.(isBinaryPacket); ok {
			v.D = binaryData.D
			isBinary = true
		}
		pay4[i] = PacketV4{PacketV3{PacketV2{Packet: v}, isBinary}}
	}
	return pay(pay4)
}
