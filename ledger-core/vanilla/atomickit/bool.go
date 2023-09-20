package atomickit

import (
	"strconv"
	"sync/atomic"
)

func NewBool(initValue bool) Bool {
	if initValue {
		return Bool{1}
	}
	return Bool{}
}

type Bool struct {
	v uint32
}

func (p *Bool) Load() bool {
	return atomic.LoadUint32(&p.v) != 0
}

func (p *Bool) Store(v bool) {
	if v {
		atomic.StoreUint32(&p.v, 1)
	} else {
		atomic.StoreUint32(&p.v, 0)
	}
}

func (p *Bool) Set() {
	atomic.StoreUint32(&p.v, 1)
}

func (p *Bool) Unset() {
	atomic.StoreUint32(&p.v, 0)
}

func (p *Bool) Swap(v bool) bool {
	if v {
		return atomic.SwapUint32(&p.v, 1) != 0
	}
	return atomic.SwapUint32(&p.v, 0) != 0
}

func (p *Bool) CompareAndFlip(old bool) bool {
	if old {
		return atomic.CompareAndSwapUint32(&p.v, 1, 0)
	}
	return atomic.CompareAndSwapUint32(&p.v, 0, 1)
}

func (p *Bool) Flip() bool {
	for v := atomic.LoadUint32(&p.v);; {
		if v != 0 {
			if atomic.CompareAndSwapUint32(&p.v, v, 0) {
				return false
			}
		} else {
			if atomic.CompareAndSwapUint32(&p.v, 0, 1) {
				return true
			}
		}
	}
}

func (p *Bool) String() string {
	return strconv.FormatBool(p.Load())
}
