//go:build gc || (eio_pay_v2 && eio_pay_v3 && eio_pay_v4)
// +build gc eio_pay_v2,eio_pay_v3,eio_pay_v4

package protocol

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// PayloadV2 is defined: https://github.com/socketio/engine.io-protocol/tree/v2
type PayloadV2 []PacketV2

type PayloadDecoderV2 struct{ r io.Reader }

var NewPayloadDecoderV2 _payloadDecoderV2 = func(r io.Reader) *PayloadDecoderV2 {
	return &PayloadDecoderV2{r: r}
}

func (dec *PayloadDecoderV2) Decode(payV2 *PayloadV2) (err error) {
	if payV2 == nil {
		payV2 = &PayloadV2{}
	}

	for {
		var pr = dec.decode(dec.r)
		var pacV2 PacketV2

		if err = NewPacketDecoderV2(pr).Decode(&pacV2); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return ErrPayloadDecode.F("v2", err)
		}

		*payV2 = append(*payV2, pacV2)
	}
}

func (dec *PayloadDecoderV2) decode(src io.Reader) io.Reader {
	var (
		b    [1]byte
		numB []byte
		numI int64
	)

	for {
		if _, err := src.Read(b[:]); err != nil {
			return src
		}
		if b[0] == ':' {
			i, err := strconv.ParseInt(string(numB), 10, 64)
			if err != nil {
				return src
			}
			numI = i
			break
		}
		numB = append(numB, b[0])
	}

	pr, pw := io.Pipe()
	go func() {
		CopyRuneN(pw, src, numI)
		pw.Close()
	}()

	return pr
}

type PayloadEncoderV2 struct{ w io.Writer }

var NewPayloadEncoderV2 _payloadEncoderV2 = func(w io.Writer) *PayloadEncoderV2 {
	return &PayloadEncoderV2{w: w}
}

func (enc *PayloadEncoderV2) Encode(payload PayloadV2) error {
	for _, packet := range payload {
		if err := enc.encode(packet); err != nil {
			return err
		}
	}
	return nil
}

func (enc *PayloadEncoderV2) encode(packet PacketV2) error {
	var buf = new(strings.Builder)
	if err := NewPacketEncoderV2(buf).Encode(packet); err != nil {
		return ErrPayloadEncode.F("v2", err)
	}
	str := buf.String()
	if _, err := fmt.Fprintf(enc.w, "%d:%s", len([]rune(str)), str); err != nil {
		return ErrPayloadEncode.F("v2", err)
	}
	return nil
}
