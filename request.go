package armie

import (
	"reflect"
	"bytes"
	"math/rand"
	"time"
	"io"
	"fmt"
	"github.com/ugorji/go/codec"
)

type Request struct {
	Method  string
	Id      uint64
	Payload []byte
}

func genID() uint64 {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	return r1.Uint64()
}

func (r *Request) DecodeArgs(types []reflect.Type) ([]interface{}, error) {
	args := make([]interface{}, len(types))
	bb := bytes.NewBuffer(r.Payload)
	dec := codec.NewDecoder(bb, &mph)
	i := 0
	var err error = nil
	for err != io.EOF && i < len(types) {
		v := reflect.New(types[i]).Elem().Interface()

		err = dec.Decode(&v)
		if err != nil && err != io.EOF {
			return nil, err
		}

		args[i] = v
		i += 1
	}
	return args, nil
}

func (r *Request) CallMethod(method interface{}) (interface{}, error) {
	types := make([]reflect.Type, 0)
	ft := reflect.TypeOf(method)

	for i := 0; i < ft.NumIn(); i++ {
		types = append(types, ft.In(i))
	}

	args, err := r.DecodeArgs(types)
	if err != nil {
		return nil, fmt.Errorf("decoding RPC arguments: %v", err)
	}

	vargs := make([]reflect.Value, 0)
	for _, arg := range args {
		vargs = append(vargs, reflect.ValueOf(arg))
	}

	var res interface{}
	results := reflect.ValueOf(method).Call(vargs)

	for _, result := range results {
		switch result.Kind() {
		case reflect.Interface:
			v := result.Interface()
			ok := false
			err, ok = v.(error)
			if !ok {
				res = v
			}
		default:
			res = result.Interface()
		}
	}

	return res, err
}

type Response struct {
	Id        uint64
	ErrString string
	Result    interface{}
	conn      *Conn
}

func (r *Response) Send(result interface{}) error {
	r.Result = result
	return encodeResponse(r.conn, r)
}

func (r *Response) Error(err string) error {
	r.ErrString = err
	return encodeResponse(r.conn, r)
}

type Event struct {
	Event   string
	Payload []byte
}

func (e *Event) Decode(v interface{}) error {
	bb := bytes.NewBuffer(e.Payload)
	dec := codec.NewDecoder(bb, &mph)

	err := dec.Decode(v)
	if err != nil {
		return err
	}

	return nil
}
