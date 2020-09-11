// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"strconv"
	"sync/atomic"
)

func NewInt32(v int32) Int32 {
	return Int32{v}
}

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
	return strconv.FormatInt(int64(p.Load()), 10)
}

func (p *Int32) SetLesser(v int32) int32 {
	for {
		switch x := p.Load(); {
		case x <= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Int32) SetGreater(v int32) int32 {
	for {
		switch x := p.Load(); {
		case x >= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}
