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
)

// Returns a level at which the unbalanced node will connect to the rightmost branch.
// It trims from left all consequent 1's starting from (bitCount) and returns a number of remaining bits.
// Doesn't panic, but result is meaningless when (maxPairLevel) is not in [0, 64] or (index) >= 1<<(maxPairLevel)
func UnbalancedBitCount(index uint, bitCount uint8) uint8 {
	if bitCount == 0 {
		return 0
	}
	return uint8(bits.Len(^(index | math.MaxInt64<<bitCount)))
}

func StackSequenceUnused(count uint) uint8 {
	if count == 0 {
		return 0
	}
	return uint8(bits.Len(count+1)) - 1
}

//func AlignedNodeCount(leafCount uint, stubbed bool) uint {
//	if stubbed {
//		depth := uint8(bits.Len(leafCount - 1))
//		// TODO stubbed node count
//		return leafCount + uint(depth - 1)
//	}
//	return leafCount + uint(StackSequenceUnused(leafCount)) - 1
//}

func AdjustStackSequenceIndexByUnused(index uint) uint {
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
