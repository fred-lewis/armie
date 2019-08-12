package armie

import (
	"bytes"
	"github.com/ugorji/go/codec"
)

var mph = codec.MsgpackHandle{}

func init() {
	mph.WriteExt = true
}

const (
	REQUEST = iota + 1
	RESPONSE
	EVENT
)

type frame struct {
	Type    uint8  `codec:"t,omitempty"`
	Method  string `codec:"m,omitempty"`
	Id      uint64 `codec:"i,omitempty"`
	Error   string `codec:"e,omitempty"`
	Payload []byte `codec:"p,omitempty"`
}

func sendFrame(conn *Conn, frm *frame) error {
	conn.connmu.Lock()
	defer conn.connmu.Unlock()
	err := conn.enc.Encode(frm)
	conn.bw.Flush()
	return err
}

func readFrame(conn *Conn) (*frame, error) {
	var frm frame
	err := conn.dec.Decode(&frm)
	if err != nil {
		return nil, err
	}
	return &frm, nil
}

func encodeRequest(conn *Conn, req *Request, args []interface{}) (*frame, error) {
	argBuf := bytes.Buffer{}
	argEnc := codec.NewEncoder(&argBuf, &mph)
	for _, arg := range args {
		argEnc.Encode(arg)
	}
	argEnc.Release()
	frm := &frame{
		Type: REQUEST,
		Method: req.Method,
		Id: req.Id,
		Payload: argBuf.Bytes(),
	}
	return frm, sendFrame(conn, frm)
}

func encodeResponse(conn *Conn, res *Response) error {
	resBuf := bytes.Buffer{}
	resEnc := codec.NewEncoder(&resBuf, &mph)
	resEnc.Encode(res.Result)
	resEnc.Release()
	frm := frame{
		Type: RESPONSE,
		Id: res.Id,
		Error: res.ErrString,
		Payload: resBuf.Bytes(),
	}
	return sendFrame(conn, &frm)
}

func encodeEvent(conn *Conn, event string, payload interface{}) error {
	msgBuf := bytes.Buffer{}
	msgEnc := codec.NewEncoder(&msgBuf, &mph)
	msgEnc.Encode(payload)
	msgEnc.Release()

	frm := frame{
		Type: EVENT,
		Method: event,
		Payload: msgBuf.Bytes(),
	}

	return sendFrame(conn, &frm)
}

func decodeResponse(frm *frame, v interface{}) error {
	bb := bytes.NewBuffer(frm.Payload)
	dec := codec.NewDecoder(bb, &mph)

	err := dec.Decode(v)
	if err != nil {
		return err
	}

	return nil
}