//go:build gc || (eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v3,eio_pay_v4

package protocol

import (
	"encoding/base64"
	"fmt"
	"io"
	"strconv"

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
			return ErrPayloadEncode.F("v4", err)
		}
	}
	return nil
}

func (enc *PayloadEncoderV3) encode(packet PacketV3) error {
	switch packet.T {
	case MessagePacket:
		if packet.IsBinary {
			if enc.IsXHR2 {
				enc.writeBinaryPacketLen(packet)

				switch data := packet.D.(type) {
				case string:
					enc.write.Bytes(packet.T.Bytes()).OnErr(ErrPayloadEncode)
					enc.write.String(data).OnErr(ErrPayloadEncode)
				case []byte:
					enc.write.Bytes(data).OnErr(ErrPayloadEncode)
				case io.Reader:
					enc.write.Copy(data).OnErr(ErrPayloadEncode)
				default:
					return fmt.Errorf("unspported type: %T", data)
				}
				return enc.write.Err()
			}

			switch data := packet.D.(type) {
			case []byte:
				b64Len := base64.StdEncoding.EncodedLen(len(data))
				enc.write.Bytes(append([]byte(strconv.Itoa(b64Len+2)), ':')).OnErr(ErrPayloadEncode)
				enc.write.Byte('b').OnErr(ErrPayloadEncode)
				enc.write.Bytes(packet.T.Bytes()).OnErr(ErrPayloadEncode)
				enc.write.Base64(base64.StdEncoding).Bytes(data).OnErr(ErrPayloadEncode)
				return enc.write.Err()
			case io.Reader:
				switch length := data.(type) {
				case useLen:
					b64Len := base64.StdEncoding.EncodedLen(length.Len())
					enc.write.Bytes(append([]byte(strconv.Itoa(b64Len+2)), ':')).OnErr(ErrPayloadEncode)
					enc.write.Byte('b').OnErr(ErrPayloadEncode)
					enc.write.Bytes(packet.T.Bytes()).OnErr(ErrPayloadEncode)
					enc.write.Base64(base64.StdEncoding).Copy(data).OnErr(ErrPayloadEncode)
					return enc.write.Err()
				}
			}
		}
	}
	return enc.PayloadEncoderV2.encode(PacketV2{Packet: packet.Packet})
}

func (enc *PayloadEncoderV3) writeBinaryPacketLen(packet PacketV3) (n int64, err error) {
	bytesInt := func(n int) []byte {
		str := strconv.Itoa(n)
		byt := make([]byte, len(str))
		for i, v := range []byte(str) {
			byt[i] = v & 0x0F
		}
		return byt
	}

	i := len(packet.T.Bytes())
	switch data := packet.D.(type) {
	case string:
		i += len(data)
		enc.write.Byte(0x00).OnErr(ErrPayloadEncode)
		enc.write.Bytes(bytesInt(i)).OnErr(ErrPayloadEncode)
	case []byte:
		i = len(data)
		enc.write.Byte(0x01).OnErr(ErrPayloadEncode)
		enc.write.Bytes(bytesInt(i)).OnErr(ErrPayloadEncode)
	case useLen:
		i = data.Len()
		enc.write.Byte(0x01).OnErr(ErrPayloadEncode)
		enc.write.Bytes(bytesInt(i)).OnErr(ErrPayloadEncode)
	}
	enc.write.Byte(0xFF).OnErr(ErrPayloadEncode)

	return int64(i), enc.write.Err()
}
