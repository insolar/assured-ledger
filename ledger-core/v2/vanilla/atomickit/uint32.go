// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"fmt"
	"sync/atomic"
)

func NewUint32(initValue uint32) Uint32 {
	return Uint32{initValue}
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
	return fmt.Sprint(p.Load())
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
