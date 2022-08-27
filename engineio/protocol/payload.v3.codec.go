//go:build gc || (eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v3,eio_pay_v4

package protocol

import (
	"io"
)

func (pay PayloadV3) PayloadVal() Payload {
	rtn := make(Payload, len(pay))
	for i, v := range pay {
		if v.IsBinary {
			v.Packet.D = isBinaryPacket{D: v.Packet.D}
		}
		rtn[i] = v.Packet
	}
	return rtn
}

func (pay *PayloadV3) PayloadRef() *Payload { return &Payload{} }

type _payloadDecoderV3 func(r io.Reader) *PayloadDecoderV3
type _payloadEncoderV3 func(w io.Writer) *PayloadEncoderV3
type _payloadReaderV3 func(pay *PayloadV3) (err error)
type _payloadWriterV3 func(pay PayloadV3) (err error)

func (pay _payloadEncoderV3) SetXHR2(isXHR2 bool) _payloadEncoderV3 {
	return func(w io.Writer) *PayloadEncoderV3 {
		enc := pay(w)
		enc.hasXHR2Support = isXHR2
		return enc
	}
}

func (pay _payloadEncoderV3) SetBinary(isBinary bool) _payloadEncoderV3 {
	return func(w io.Writer) *PayloadEncoderV3 {
		enc := pay(w)
		enc.hasBinarySupport = isBinary
		return enc
	}
}

func (pay _payloadDecoderV3) SetXHR2(isXHR2 bool) _payloadDecoderV3 {
	return func(r io.Reader) *PayloadDecoderV3 {
		dec := pay(r)
		dec.hasXHR2Support = isXHR2
		return dec
	}
}

func (pay _payloadDecoderV3) SetBinary(isBinary bool) _payloadDecoderV3 {
	return func(r io.Reader) *PayloadDecoderV3 {
		dec := pay(r)
		dec.hasBinarySupport = isBinary
		return dec
	}
}

func (pay _payloadDecoderV3) From(r io.Reader) PayloadReader { return _payloadReaderV3(pay(r).Decode) }
func (pay _payloadEncoderV3) To(w io.Writer) PayloadWriter   { return _payloadWriterV3(pay(w).Encode) }
func (pay _payloadReaderV3) ReadPayload(payload PayloadRef) (err error) {
	var pay3 PayloadV3
	if err = pay(&pay3); err != nil {
		return err
	}

	switch pay := payload.(type) {
	case *Payload:
		payRef := make([]Packet, len(pay3))
		for i, v := range pay3 {
			payRef[i] = v.Packet
		}
		*pay = payRef
	case *PayloadV3:
		*pay = pay3
	}

	return nil
}
func (pay _payloadWriterV3) WritePayload(payload PayloadVal) error {
	pay3 := make(PayloadV3, len(payload.PayloadVal()))
	for i, v := range payload.PayloadVal() {
		var isBinary bool
		if binaryData, ok := v.D.(isBinaryPacket); ok {
			v.D = binaryData.D
			isBinary = true
		}
		pay3[i] = PacketV3{PacketV2{Packet: v}, isBinary}
	}
	return pay(pay3)
}
