// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"fmt"
	"sync/atomic"
)

func NewUint64(initValue uint64) Uint64 {
	return Uint64{initValue}
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
	return fmt.Sprint(p.Load())
}
