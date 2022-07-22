package protocol

// packetNS represents the Type part of a SocketIO packet
type packetType byte

func (x packetType) Len() int {
	switch {
	case x < 10:
		return 1
	case x < 100:
		return 2
	}
	return 3
}

// Byte returns the Type as the underlining byte type
func (x packetType) Byte() byte { return byte(x) }

// Read reads the Type string to the p byte slice as an ASCII value.
func (x packetType) Read(p []byte) (n int, err error) {
	return copy(p, []byte{byte(x + '0')}), nil // the x + '0' is a trick where we add the byte to the string 0 to get the ASCII byte representation
}

// Write writes the data coming from p to the underlining byte data type, converting from a string to an int8.
func (x *packetType) Write(p []byte) (n int, err error) {
	// no need to plan for short writes... as this is one byte... and the first one at that...
	*x = packetType(p[0] & 0x0F)
	return 1, nil
}
