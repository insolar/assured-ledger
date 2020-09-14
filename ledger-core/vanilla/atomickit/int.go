// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"math/bits"
	"strconv"
	"sync/atomic"
	"unsafe"
)

func NewInt(v int) Int {
	return Int{v}
}

type Int struct {
	v int
}

func (p *Int) ptr32() *int32 {
	return (*int32)(unsafe.Pointer(&p.v))
}

func (p *Int) ptr64() *int64 {
	return (*int64)(unsafe.Pointer(&p.v))
}

func (p *Int) Load() int {
	if bits.UintSize == 32 {
		return int(atomic.LoadInt32(p.ptr32()))
	}
	return int(atomic.LoadInt64(p.ptr64()))
}

func (p *Int) Store(v int) {
	if bits.UintSize == 32 {
		atomic.StoreInt32(p.ptr32(), int32(v))
		return
	}
	atomic.StoreInt64(p.ptr64(), int64(v))
}

func (p *Int) Swap(v int) int {
	if bits.UintSize == 32 {
		return int(atomic.SwapInt32(p.ptr32(), int32(v)))
	}
	return int(atomic.SwapInt64(p.ptr64(), int64(v)))
}

func (p *Int) CompareAndSwap(old, new int) bool {
	if bits.UintSize == 32 {
		return atomic.CompareAndSwapInt32(p.ptr32(), int32(old), int32(new))
	}
	return atomic.CompareAndSwapInt64(p.ptr64(), int64(old), int64(new))
}

func (p *Int) Add(v int) int {
	if bits.UintSize == 32 {
		return int(atomic.AddInt32(p.ptr32(), int32(v)))
	}
	return int(atomic.AddInt64(p.ptr64(), int64(v)))
}

func (p *Int) Sub(v int) int {
	return p.Add(-v)
}

func (p *Int) String() string {
	return strconv.FormatInt(int64(p.Load()), 10)
}

func (p *Int) SetLesser(v int) int {
	for {
		switch x := p.Load(); {
		case x <= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Int) SetGreater(v int) int {
	for {
		switch x := p.Load(); {
		case x >= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}
