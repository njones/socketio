package protocol

import "bytes"

// packetNS represents the Namespace part of a SocketIO packet
type packetNS string

func (x packetNS) Len() int {
	if x == "/" || x == "" {
		return 0
	}
	return len(x)
}

// Read reads the Namespace string to the p byte slice. If the p byte slice is
// not big enough then it errors with a buffer of the rest of the data. The error
// can be compared to the error ErrShortRead.
func (x packetNS) Read(p []byte) (n int, err error) {
	if len(x) == 0 {
		return
	}

	if x == "/" {
		return // a single "/" is the same as empty
	}

	if n = copy(p, x); n < len(x) {
		return n, ErrReadUseBuffer.BufferF("namespace", []byte(x)[n:], ErrShortRead)
	}

	return n, nil
}

// Write writes the data coming from p to the underlining string data type.
func (x *packetNS) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		if len(*x) == 0 {
			*x = "/"
		}
		return
	}

	var data []byte
	if x != nil && len(*x) > 0 { // this means we have a short write, so pick up where we left off...
		data = []byte(*x)
	}

	var size = len(data)
	for i, val := range p {
		if i == 0 && val != '/' && len(data) == 0 {
			*x = packetNS("/")
			return 0, nil
		}
		switch val {
		case ',':
			// Fix github:#61 Removing query parameters from the Namespace
			if idxQ := bytes.Index(data, []byte{'?'}); idxQ > -1 {
				data = data[:idxQ]
			}
			*x = packetNS(string(data))
			return i + 1, nil
		}
		data = append(data, val)
	}

	if (len(data) - size) == len(p) {
		readSize := len(data)
		// Fix github:#61 Removing query parameters from the Namespace
		if idxQ := bytes.Index(data, []byte{'?'}); idxQ > -1 {
			data = data[:idxQ]
		}
		*x = packetNS(string(data))
		return readSize - size, nil
	}

	return len(data) - size, ErrShortWrite
}
