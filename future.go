package armie

import "sync"

//
//  RPC future for awaiting responses to RMI requests
//
type Future struct {
	wg  sync.WaitGroup
	res *frame
	err error
}

func newFuture() *Future {
	f := &Future{}
	f.wg.Add(1)

	return f
}


func (f *Future) complete(res *frame) {
	f.res = res
	f.wg.Done()
}

func (f *Future) error(err error) {
	f.err = err
	f.wg.Done()
}

//
// Await the response or error.  If the error returned is not-nil,
// the result will be nil.
//
func (f *Future) GetResult(res interface{}) error {
	f.wg.Wait()
	if f.err != nil {
		return f.err
	}

	var err error = nil
	if res != nil {
		err = decodeResponse(f.res, res)
	}

	return err
}
