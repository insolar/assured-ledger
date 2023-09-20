package atomickit

import (
	"strconv"
	"sync/atomic"
)

func NewUint64(v uint64) Uint64 {
	return Uint64{v}
}

type Uint64 struct {
	v uint64
}

func (p *Uint64) Load() uint64 {
	return atomic.LoadUint64(&p.v)
}

func (p *Uint64) Store(v uint64) {
	atomic.StoreUint64(&p.v, v)
}

func (p *Uint64) Swap(v uint64) uint64 {
	return atomic.SwapUint64(&p.v, v)
}

func (p *Uint64) CompareAndSwap(old, new uint64) bool {
	return atomic.CompareAndSwapUint64(&p.v, old, new)
}

func (p *Uint64) Add(v uint64) uint64 {
	return atomic.AddUint64(&p.v, v)
}

func (p *Uint64) Sub(v uint64) uint64 {
	return p.Add(^(v - 1))
}

func (p *Uint64) String() string {
	return strconv.FormatUint(p.Load(), 10)
}

func (p *Uint64) SetBits(v uint64) uint64 {
	for {
		switch x := p.Load(); {
		case x & v == v:
			return x
		case p.CompareAndSwap(x, x|v):
			return x|v
		}
	}
}

func (p *Uint64) TrySetBits(v uint64, all bool) bool {
	for {
		x := p.Load()
		switch {
		case x & v == 0:
		case all:
			return false
		case x & v == v:
			return false
		}
		if p.CompareAndSwap(x, x|v) {
			return true
		}
	}
}

func (p *Uint64) UnsetBits(v uint64) uint64 {
	for {
		switch x := p.Load(); {
		case x & v == 0:
			return x
		case p.CompareAndSwap(x, x&^v):
			return x&^v
		}
	}
}

func (p *Uint64) TryUnsetBits(v uint64, all bool) bool {
	for {
		x := p.Load()
		switch {
		case x & v == 0:
			return false
		case x & v == v:
		case all:
			return false
		}
		if p.CompareAndSwap(x, x&^v) {
			return true
		}
	}
}

func (p *Uint64) CompareAndSub(v uint64) bool {
	for {
		switch x := p.Load(); {
		case x < v:
			return false
		case p.CompareAndSwap(x, x-v):
			return true
		}
	}
}

func (p *Uint64) SetLesser(v uint64) uint64 {
	for {
		switch x := p.Load(); {
		case x <= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Uint64) SetGreater(v uint64) uint64 {
	for {
		switch x := p.Load(); {
		case x >= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Uint64) CompareAndSetBits(maskOut, deny, v uint64) bool {
	for {
		x := p.Load()
		x2 := x &^ maskOut
		switch {
		case x2 & deny != 0:
			return false
		case x2 & v != 0:
			return false
		}
		if p.CompareAndSwap(x, x|v) {
			return true
		}
	}
}
