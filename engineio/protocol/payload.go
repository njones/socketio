package protocol

import (
	"io"

	rw "github.com/njones/socketio/internal/readwriter"
)

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

type reader struct {
	*rw.Reader
	err error
}

// func (rp *readPayload) ReadString(delim byte) (str string) {
// 	if rp.err != nil {
// 		return ""
// 	}
// 	str, rp.err = rp.r.ReadString(delim)
// 	return str
// }

// func (rp *readPayload) StrToInt(str string, cutset string) (n int64) {
// 	strInt := strings.Trim(str, cutset)
// 	n, rp.err = strconv.ParseInt(strInt, 10, 64)
// 	return n
// }
