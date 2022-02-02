package transport

func packetToSocket(pac packet) Socket {
	return Socket{
		Type:      pac.GetType(),
		Namespace: pac.GetNamespace(),
		AckID:     pac.GetAckID(),
		Data:      pac.GetData(),
	}
}
