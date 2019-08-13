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

//
// An RMI request containing encoded arguments and a method name
//
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

//
// Decode arguments using reflect.Types.  In most cases (true RMI) CallMethod()
// will be more convenient (it uses decodeArgs internally base on the target
// method arguments)
//
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

//
// Decode request arguments to match the method args, and call the method.
// The method can return an error and, at most one other object, which will
// in-turn be returned by CallMethod.
//
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

//
// Response is passed to a RequestHandler to allow the RequestHandler
// to send a result or an error.
//
type Response struct {
	Id        uint64
	ErrString string
	Result    interface{}
	conn      *Conn
}

//
// Send the given result.  After a result is sent, the Response is
// no longer usable.
//
func (r *Response) Send(result interface{}) error {
	r.Result = result
	return encodeResponse(r.conn, r)
}

//
// Send the given error.  After an error is sent, the Response is
// no longer usable.
//
func (r *Response) Error(err string) error {
	r.ErrString = err
	return encodeResponse(r.conn, r)
}

//
// An event containing an event name and an encoded payload.
//
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
