package atomickit

import (
	"math/bits"
	"strconv"
	"sync/atomic"
	"unsafe"
)

func NewUint(v uint) Uint {
	return Uint{v}
}

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
	return strconv.FormatUint(uint64(p.Load()), 10)
}

func (p *Uint) SetBits(v uint) uint {
	for {
		switch x := p.Load(); {
		case x & v == v:
			return x
		case p.CompareAndSwap(x, x|v):
			return x|v
		}
	}
}

func (p *Uint) TrySetBits(v uint, all bool) bool {
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

func (p *Uint) UnsetBits(v uint) uint {
	for {
		switch x := p.Load(); {
		case x & v == 0:
			return x
		case p.CompareAndSwap(x, x&^v):
			return x&^v
		}
	}
}

func (p *Uint) TryUnsetBits(v uint, all bool) bool {
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

func (p *Uint) CompareAndSub(v uint) bool {
	for {
		switch x := p.Load(); {
		case x < v:
			return false
		case p.CompareAndSwap(x, x-v):
			return true
		}
	}
}

func (p *Uint) SetLesser(v uint) uint {
	for {
		switch x := p.Load(); {
		case x <= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Uint) SetGreater(v uint) uint {
	for {
		switch x := p.Load(); {
		case x >= v:
			return x
		case p.CompareAndSwap(x, v):
			return v
		}
	}
}

func (p *Uint) CompareAndSetBits(maskOut, deny, v uint) bool {
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
