package protocol

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

func (rdr *reader) Peek(n int) []byte {
	if rdr.IsErr() {
		return nil
	}

	b, err := rdr.Bufio().Peek(n)
	rdr.SetErr(err)
	return b
}

func (rdr *reader) readByte() (b byte) {
	if rdr.err != nil {
		return 0
	}

	b, rdr.err = rdr.Bufio().ReadByte()
	return b
}

func (rdr *reader) readBinaryPacketLen() (n int64) {
	if rdr.err != nil {
		return 0
	}

	var data []byte
	data, rdr.err = rdr.Bufio().ReadBytes(0xFF)
	if rdr.err != nil {
		return 0
	}

	data = bytes.TrimRight(data, "\xFF")
	for i, v := range data {
		data[i] = []byte(fmt.Sprintf("%d", v))[0]
	}

	n, rdr.err = strconv.ParseInt(string(data), 10, 64)
	if rdr.err != nil {
		return 0
	}

	return n
}

func (rdr *reader) decodeXHR2(payload *PayloadV3) error {

	for rdr.IsNotErr() {
		b := rdr.readByte()

		var isBinary = (b == 0x01)

		n := rdr.readBinaryPacketLen()

		var packet PacketV3
		if isBinary {
			packet.T = MessagePacket
			packet.IsBinary = true
		}

		if rdr.IsNotErr() && rdr.ConditionalErr(NewPacketDecoderV3(io.LimitReader(rdr, n)).Decode(&packet)).IsNotErr() {
			*payload = append(*payload, packet)
		}
	}

	return rdr.ConvertErr(io.EOF, nil).Err()
}

// Writer methods

func (wtr *writer) writeBinaryPacketLen(n int) {
	for _, r := range []byte(strconv.Itoa(n)) {
		wtr.Write([]byte{r - '0'})
	}
}

func (wtr *writer) encodeXHR2(packet PacketV3) error {
	switch {
	case packet.IsBinary:
		wtr.Write([]byte{0x01})
	default:
		wtr.Write([]byte{0x00})
	}

	switch val := packet.D.(type) {
	case string:
		wtr.writeBinaryPacketLen(len([]byte(val)) + 1) // +1 for the message type
		wtr.Write([]byte{0xFF})
		wtr.Write(packet.T.Bytes())
		wtr.Write([]byte(val))
	case io.Reader:
		wtr.writeBinaryPacketLen(packet.Len() - 1) // -1 for the message type
		wtr.Write([]byte{0xFF})
		wtr.Copy(val)
	}

	return wtr.Err()
}
