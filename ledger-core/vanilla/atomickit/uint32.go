// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"strconv"
	"sync/atomic"
)

func NewUint32(v uint32) Uint32 {
	return Uint32{v}
}

type Uint32 struct {
	v uint32
}

func (p *Uint32) Load() uint32 {
	return atomic.LoadUint32(&p.v)
}

func (p *Uint32) Store(v uint32) {
	atomic.StoreUint32(&p.v, v)
}

func (p *Uint32) Swap(v uint32) uint32 {
	return atomic.SwapUint32(&p.v, v)
}

func (p *Uint32) CompareAndSwap(old, new uint32) bool {
	return atomic.CompareAndSwapUint32(&p.v, old, new)
}

func (p *Uint32) Add(v uint32) uint32 {
	return atomic.AddUint32(&p.v, v)
}

func (p *Uint32) Sub(v uint32) uint32 {
	return p.Add(^(v - 1))
}

func (p *Uint32) String() string {
	return strconv.FormatUint(uint64(p.Load()), 10)
}

func (p *Uint32) SetBits(v uint32) uint32 {
	for {
		switch x := p.Load(); {
		case x & v == v:
			return x
		case p.CompareAndSwap(x, x|v):
			return x|v
		}
	}
}

func (p *Uint32) TrySetBits(v uint32, all bool) bool {
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

func (p *Uint32) UnsetBits(v uint32) uint32 {
	for {
		switch x := p.Load(); {
		case x & v == 0:
			return x
		case p.CompareAndSwap(x, x&^v):
			return x&^v
		}
	}
}

func (p *Uint32) TryUnsetBits(v uint32, all bool) bool {
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

func (p *Uint32) CompareAndSub(v uint32) bool {
	for {
		switch x := p.Load(); {
		case x < v:
			return false
		case p.CompareAndSwap(x, x-v):
			return true
		}
	}
}

func (p *Uint32) SetLesser(v uint32) uint32 {
	for {
		switch x := p.Load(); {
		case x <= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Uint32) SetGreater(v uint32) uint32 {
	for {
		switch x := p.Load(); {
		case x >= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Uint32) CompareAndSetBits(maskOut, deny, v uint32) bool {
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
