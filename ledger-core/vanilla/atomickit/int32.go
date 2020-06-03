// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"fmt"
	"sync/atomic"
)

type Int32 struct {
	v int32
}

func (p *Int32) Load() int32 {
	return atomic.LoadInt32(&p.v)
}

func (p *Int32) Store(v int32) {
	atomic.StoreInt32(&p.v, v)
}

func (p *Int32) Swap(v int32) int32 {
	return atomic.SwapInt32(&p.v, v)
}

func (p *Int32) CompareAndSwap(old, new int32) bool {
	return atomic.CompareAndSwapInt32(&p.v, old, new)
}

func (p *Int32) Add(v int32) int32 {
	return atomic.AddInt32(&p.v, v)
}

func (p *Int32) Sub(v int32) int32 {
	return p.Add(-v)
}

func (p *Int32) String() string {
	return fmt.Sprint(p.Load())
}
