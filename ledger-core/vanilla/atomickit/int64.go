package atomickit

import (
	"strconv"
	"sync/atomic"
)

func NewInt64(v int64) Int64 {
	return Int64{v}
}

type Int64 struct {
	v int64
}

func (p *Int64) Load() int64 {
	return atomic.LoadInt64(&p.v)
}

func (p *Int64) Store(v int64) {
	atomic.StoreInt64(&p.v, v)
}

func (p *Int64) Swap(v int64) int64 {
	return atomic.SwapInt64(&p.v, v)
}

func (p *Int64) CompareAndSwap(old, new int64) bool {
	return atomic.CompareAndSwapInt64(&p.v, old, new)
}

func (p *Int64) Add(v int64) int64 {
	return atomic.AddInt64(&p.v, v)
}

func (p *Int64) Sub(v int64) int64 {
	return p.Add(-v)
}

func (p *Int64) String() string {
	return strconv.FormatInt(p.Load(), 10)
}

func (p *Int64) SetLesser(v int64) int64 {
	for {
		switch x := p.Load(); {
		case x <= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Int64) SetGreater(v int64) int64 {
	for {
		switch x := p.Load(); {
		case x >= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}
