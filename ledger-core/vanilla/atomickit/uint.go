// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"fmt"
	"math/bits"
	"sync/atomic"
	"unsafe"
)

type Uint struct {
	v uint
}

func (p *Uint) ptr32() *uint32 {
	return (*uint32)(unsafe.Pointer(&p.v))
}

func (p *Uint) ptr64() *uint64 {
	return (*uint64)(unsafe.Pointer(&p.v))
}

func (p *Uint) Load() uint {
	if bits.UintSize == 32 {
		return uint(atomic.LoadUint32(p.ptr32()))
	}
	return uint(atomic.LoadUint64(p.ptr64()))
}

func (p *Uint) Store(v uint) {
	if bits.UintSize == 32 {
		atomic.StoreUint32(p.ptr32(), uint32(v))
		return
	}
	atomic.StoreUint64(p.ptr64(), uint64(v))
}

func (p *Uint) Swap(v uint) uint {
	if bits.UintSize == 32 {
		return uint(atomic.SwapUint32(p.ptr32(), uint32(v)))
	}
	return uint(atomic.SwapUint64(p.ptr64(), uint64(v)))
}

func (p *Uint) CompareAndSwap(old, new uint) bool {
	if bits.UintSize == 32 {
		return atomic.CompareAndSwapUint32(p.ptr32(), uint32(old), uint32(new))
	}
	return atomic.CompareAndSwapUint64(p.ptr64(), uint64(old), uint64(new))
}

func (p *Uint) Add(v uint) uint {
	if bits.UintSize == 32 {
		return uint(atomic.AddUint32(p.ptr32(), uint32(v)))
	}
	return uint(atomic.AddUint64(p.ptr64(), uint64(v)))
}

func (p *Uint) Sub(v uint) uint {
	return p.Add(^(v - 1))
}

func (p *Uint) String() string {
	return fmt.Sprint(p.Load())
}
