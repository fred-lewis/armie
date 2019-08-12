package armie

import "sync"

//
//  RPC future for awaiting responses to ARMIE requests
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
