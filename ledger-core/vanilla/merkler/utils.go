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
