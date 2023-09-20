package atomickit

import (
	"runtime"
	"sync/atomic"
)

type OnceFlag struct {
	done int32
}

func (p *OnceFlag) IsSet() bool {
	return atomic.LoadInt32(&p.done) == 1
}

func (p *OnceFlag) Set() bool {
	return atomic.CompareAndSwapInt32(&p.done, 0, 1)
}

func (p *OnceFlag) DoSet(f func()) bool {
	if !atomic.CompareAndSwapInt32(&p.done, 0, -1) {
		return false
	}
	p.doSlow(f)
	return true
}

func (p *OnceFlag) DoSpin(f func()) bool {
	if !atomic.CompareAndSwapInt32(&p.done, 0, -1) {
		for !p.IsSet() {
			runtime.Gosched()
		}
		return false
	}
	p.doSlow(f)
	return true
}

func (p *OnceFlag) doSlow(f func()) {
	defer atomic.StoreInt32(&p.done, 1)
	f()
}

