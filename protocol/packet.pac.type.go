package protocol

type packetType byte

func (x packetType) Byte() byte { return byte(x) }

func (x packetType) Read(p []byte) (n int, err error) {
	return copy(p, []byte{byte(x + '0')}), nil
}

func (x *packetType) Write(p []byte) (n int, err error) {
	// no need to plan for short writes... as this is one byte... and the first one at that...
	*x = packetType(p[0] & 0x0F)
	return 1, nil
}
