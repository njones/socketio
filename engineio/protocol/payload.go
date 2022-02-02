package protocol

import "io"

type Payload []Packet

func (pay Payload) PayloadVal() Payload   { return pay }
func (pay *Payload) PayloadRef() *Payload { return pay }

type (
	PayloadEncoder interface{ To(io.Writer) PayloadWriter }
	PayloadDecoder interface{ From(io.Reader) PayloadReader }

	PayloadVal interface{ PayloadVal() Payload }
	PayloadRef interface{ PayloadRef() *Payload }

	PayloadWriter interface{ WritePayload(PayloadVal) error }
	PayloadReader interface{ ReadPayload(PayloadRef) error }
)
