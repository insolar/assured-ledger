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

package merkler

import (
	"math"
	"math/bits"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/args"
)

func NewPathBuilder(count uint, stubbed bool) PathBuilder {
	if count == 0 {
		panic("illegal value")
	}

	depth := uint8(bits.Len(count - 1))

	balancedCount := count
	if !args.IsPowerOfTwo(count) {
		balancedCount = 1 << (depth - 1)
	}

	return PathBuilder{count, balancedCount, depth, stubbed}
}

type PathBuilder struct {
	count         uint
	balancedCount uint
	depth         uint8
	stubbed       bool
}

func (v PathBuilder) WalkFor(index uint, siblingFn func(traceIndex uint)) {
	if index >= v.count {
		panic("illegal value")
	}
	for level := uint8(0); level < v.depth; level++ {
		tIndex := prefixToAbsIndex(index^1, level)
		switch {
		case tIndex < v.count<<1:
			tIndex = indexSkip(tIndex)
		case v.stubbed:
			// TODO stubbed pos
		default:
			continue
		}
		siblingFn(tIndex)
		index >>= 1
	}
}

func prefixToAbsIndex(prefix uint, level uint8) uint {
	switch level {
	case 0:
		return prefix << 1
	case 1:
		return prefix<<2 | 3
	}
	index := ^(math.MaxUint64 << level) | prefix<<level
	index <<= 1
	index += 1<<level - 1
	return index
}

func indexSkip(index uint) uint {
	switch index {
	case 0:
		return 0
	case 1:
		panic("illegal value")
	}
	b := uint(bits.Len(index))
	switch skip := uint(1)<<b - 3; {
	case index < skip:
		index++
	case index == skip:
		panic("illegal value")
	}
	index -= b - 1
	return index
}
