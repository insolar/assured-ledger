//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package atomickit

import (
	"fmt"
	"math/bits"
	"sync/atomic"
	"unsafe"
)

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
	return fmt.Sprint(p.Load())
}
