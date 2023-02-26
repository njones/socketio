package protocol

import "strconv"

// packetAckID represents the AckID part of a socket.io packet
type packetAckID uint64

func (x packetAckID) Len() int {
	if x == 0 {
		return 0
	}
	return len(strconv.FormatUint(uint64(x), 10))
}

// Read reads the AckID to the p byte slice. If the p byte slice is not big
// enough then it errors with a buffer of the rest of the data. The error
// can be compared to the error ErrShortRead.
func (x packetAckID) Read(p []byte) (n int, err error) {
	if x == 0 {
		return
	}

	numStr := strconv.FormatUint(uint64(x), 10)
	n = copy(p, []byte(numStr))

	if n < len(numStr) {
		return n, ErrReadUseBuffer.BufferF("AckID", []byte(numStr)[n:], ErrShortRead)
	}

	return n, nil
}

// Write writes the data coming from p to the underlining uint64 data type.
func (x *packetAckID) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return
	}

	var val byte
	var data []byte
	if x != nil && *x != 0 { // this means we have a short write, so pick up where we left off...
		data = []byte(strconv.FormatUint(uint64(*x), 10))
	}

AckIDNumber:
	for _, val = range p {
		n++
		switch val {
		case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
			data = append(data, val)
		case '[', '{', '"':
			n-- // because we don't want to keep this character in our output, let it live for another day...
			if n == 0 {
				return 0, nil
			}
			break AckIDNumber
		}
	}

	i, err := strconv.ParseUint(string(data), 10, 64)
	*x = packetAckID(i)

	return n, err
}
