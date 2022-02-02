package protocol

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

type PayloadDecoderV4 struct {
	*PayloadDecoderV3

	buf *bufio.Reader
}

var NewPayloadDecoderV4 _payloadDecoderV4 = func(r io.Reader) *PayloadDecoderV4 {
	dec := &PayloadDecoderV4{
		PayloadDecoderV3: &PayloadDecoderV3{
			PayloadDecoderV2: &PayloadDecoderV2{r: r},
		},
	}
	dec.buf = bufio.NewReader(dec.r) // pull everything through here...
	return dec
}

func (dec *PayloadDecoderV4) Decode(payload *PayloadV4) (err error) {
	if payload == nil {
		payload = &PayloadV4{}
	}

	decErr := make(chan error, 1)

	for {
		var r io.Reader
		var pacV4 PacketV4
		var pacType [1]byte

		pr, pw := io.Pipe()
		go func() {
			decErr <- dec.decode(pw, dec.buf)
			pw.Close()
		}()

		pr.Read(pacType[:])
		if pacType[0] == 'b' {
			pacV4.IsBinary = true
			r = io.MultiReader(bytes.NewReader(pacType[:]), base64.NewDecoder(base64.StdEncoding, pr))
		} else {
			r = io.MultiReader(bytes.NewReader(pacType[:]), pr)
		}

		if err = NewPacketDecoderV3(r).Decode(&pacV4); err != nil {
			if errors.Is(err, io.EOF) {
				return nil // where we nornally finish...
			}
			return ErrPayloadDecode.F("v4", err)
		}

		if err := <-decErr; err != nil && !errors.Is(err, io.EOF) {
			return err // no need to wrap here, the error is wrapped in the decode() method
		}

		*payload = append(*payload, pacV4)

	}
}

var cutRecSep = string(rune(RecordSeparator))

func (dec *PayloadDecoderV4) decode(dst io.Writer, src io.Reader) error {
	if buf, ok := src.(*bufio.Reader); ok {
		data, err := buf.ReadBytes(RecordSeparator)
		if err != nil && !errors.Is(err, io.EOF) {
			return ErrPayloadDecode.F("v4", err)
		}

		if _, err := dst.Write(bytes.TrimRight(data, cutRecSep)); err != nil && !errors.Is(err, io.EOF) {
			return ErrPayloadDecode.F("v4", err)
		}
		return nil
	}

	return ErrBuffReaderRequired
}

// PayloadV4 is defined: https://github.com/socketio/engine.io-protocol/tree/39c138a1b54567c18f26da09ad6c07499dc0f0a0
type PayloadV4 []PacketV4
type PacketV4 = PacketV3

const RecordSeparator = 0x1e

type PayloadEncoderV4 struct{ *PayloadEncoderV3 }

var NewPayloadEncoderV4 _payloadEncoderV4 = func(w io.Writer) *PayloadEncoderV4 {
	return &PayloadEncoderV4{PayloadEncoderV3: &PayloadEncoderV3{
		PayloadEncoderV2: &PayloadEncoderV2{w: w},
	}}
}

func (enc *PayloadEncoderV4) Encode(payload PayloadV4) error {
	for i, packet := range payload {
		if i > 0 {
			enc.w.Write([]byte{RecordSeparator})
		}
		if err := enc.encode(packet); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return ErrPayloadEncode.F("v4", err)
		}
	}
	return nil
}

func (enc *PayloadEncoderV4) encode(packet PacketV4) error {
	switch packet.T {
	case MessagePacket: // the only place for binary data in v4
		var data = new(strings.Builder)
		switch value := packet.D.(type) {
		case []byte:
			b64 := base64.NewEncoder(base64.StdEncoding, data)
			b64.Write(value)
			b64.Close()
			if _, err := fmt.Fprintf(enc.w, "b%s", data.String()); err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return ErrPayloadEncode.F("v4", err)
			}
			return nil
		}
		fallthrough
	case OpenPacket, ClosePacket, PingPacket, PongPacket, UpgradePacket, NoopPacket:
		if err := NewPacketEncoderV3(enc.w).Encode(packet); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return ErrPayloadEncode.F("v4", err)
		}
	}
	return nil
}
