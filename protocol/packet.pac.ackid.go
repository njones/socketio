package protocol

import "strconv"

type packetAckID uint64

func (x packetAckID) Read(p []byte) (n int, err error) {
	if x == 0 {
		return
	}

	numStr := strconv.FormatUint(uint64(x), 10)
	n = copy(p, []byte(numStr))

	if n < len(numStr) {
		return n, PacketError{str: "buffer AckID for read", buffer: []byte(numStr)[n:], errs: []error{ErrShortRead}}
	}

	return n, nil
}

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
		case '[', '"':
			n-- // becuase we don't want to keep this character in our output, let it live for another day...
			if n == 0 {
				return n, nil // ErrNoAckID
			}
			break AckIDNumber
		}
	}

	i, err := strconv.ParseUint(string(data), 10, 64)
	*x = packetAckID(i)

	return n, err
}
