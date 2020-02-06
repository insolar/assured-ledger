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
