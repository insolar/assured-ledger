package merkler

import (
	"math"
	"math/bits"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
)

// PathBuilder allows to build a merkle-proof by a linear hashing log produced by StackedCalculator.
// PathBuilder should be initialized by NewPathBuilder, then use WalkFor to get positions of the hashing log entries
// required to be included into a merkle-proof.
//
// Complexity (by a number of hashes/leafs):
//  - NewPathBuilder(leafCount, stubbed) is O(k * log(leafCount)), k ~ 1 when !stubbed and k ~ 2 otherwise
//  - WalkFor(index) is O(log(leafCount))
//  - Memory is O(log(leafCount))

type PathBuilder struct {
	count   uint
	levels  []pathLevel
	stubbed bool
}

type pathLevel struct {
	bitMask    uint
	nodeValueL uint
	nodeValueR uint
}

func NewPathBuilder(leafCount uint, stubbed bool) PathBuilder {
	switch {
	case leafCount > 2:
		// jump to the below
	case leafCount == 0:
		panic("illegal value")
	default:
		return PathBuilder{leafCount, nil, stubbed}
	}

	depth := uint8(bits.Len(leafCount - 1))
	levels := make([]pathLevel, depth-1)

	for i := range levels {
		bitMask := uint(2) << uint8(i)
		levels[i].bitMask = bitMask
	}

	switch {
	case args.IsPowerOfTwo(leafCount):
		perfectPath(leafCount, levels)
	case stubbed:
		stubbedPath(leafCount, levels)
	default:
		unstubbedPath(leafCount, levels)
	}

	return PathBuilder{leafCount, levels, stubbed}
}

func perfectPath(leafCount uint, levels []pathLevel) {
	nodeCount := leafCount
	for i := 1; i < len(levels); i++ {
		levels[i].nodeValueR = nodeCount
		nodeCount++
	}
}

func unstubbedPath(leafCount uint, levels []pathLevel) {
	lastLeafIndex := leafCount - 1

	invertMask := ^(uint(math.MaxUint64&^1) << UnbalancedBitCount(lastLeafIndex, uint8(len(levels)+1)))
	invertedLastBalanced := lastLeafIndex ^ invertMask // | invertMask>>1
	//if invertedLastBalanced >= leafCount {
	//	panic("illegal state")
	//}

	lastRightmost := lastLeafIndex
	nodeIndex := leafCount

	for i, level := range levels {
		if lastLeafIndex&level.bitMask == 0 {
			levels[i].nodeValueL = lastRightmost

			leftGapSibling := invertedLastBalanced ^ level.bitMask
			//if leftGapSibling >= leafCount {
			//	panic("illegal state")
			//}
			if indexR := siblingIndex(leftGapSibling, level.bitMask); indexR >= leafCount {
				levels[i].nodeValueR = nodeIndex
				nodeIndex++
			}
			continue
		}

		indexL := siblingIndex(lastLeafIndex, level.bitMask)
		if indexL >= leafCount {
			levels[i].nodeValueL = nodeIndex
			nodeIndex++
		}

		levels[i].nodeValueR = lastRightmost
		lastRightmost = nodeIndex
		nodeIndex++
	}

	if nodeCount := leafCount + uint(StackSequenceUnused(leafCount)) - 1; nodeCount != nodeIndex {
		panic("illegal state")
	}
}

func stubbedPath(leafCount uint, levels []pathLevel) {
	lastLeafIndex := leafCount - 1

	invertMask := ^(uint(math.MaxUint64&^1) << UnbalancedBitCount(lastLeafIndex, uint8(len(levels)+1)))
	invertedLastBalanced := lastLeafIndex ^ invertMask // | invertMask>>1
	//if invertedLastBalanced >= leafCount {
	//	panic("illegal state")
	//}

	nodeIndex := leafCount

	for i, level := range levels {
		if i == 0 && leafCount&1 == 0 {
			continue
		}

		if lastLeafIndex&level.bitMask == 0 {
			leftGapSibling := invertedLastBalanced ^ level.bitMask
			//if leftGapSibling >= leafCount {
			//	panic("illegal state")
			//}
			if indexR := siblingIndex(leftGapSibling, level.bitMask); indexR >= leafCount {
				levels[i].nodeValueR = nodeIndex
				nodeIndex++
			}

			levels[i].nodeValueL = nodeIndex
			nodeIndex++
			continue
		}

		indexL := siblingIndex(lastLeafIndex, level.bitMask)
		if indexL >= leafCount {
			levels[i].nodeValueL = nodeIndex
			nodeIndex++
		}

		levels[i].nodeValueR = nodeIndex
		nodeIndex++
	}

	// TODO check count
}

const StubNodeIndex = 0

// Is called in a sequence of merkle-proof items, starting from leafs.
// (index) is an index of a relevant entry of StackCalculator hashing log.
// (isLeaf) indicates is this should be a leaf value or a node value.
// (isRight) indicates position of this entry for a hashing operation. Either left or right.
//
// NB! Leaf index is [0, leafCount-1], node index is [1, leafCount+N], node index = 0 means stub value.
//
type PathEntryFunc func(index uint, isLeaf, isRight bool)

// For the given (index) WalkFor will call (nodeFn) for each level of tree that has to be included into a merkle-proof, starting from leafs.
// The (nodeFn) is called with a relevant index of a hash value in the merkler log, and with flags about the value - leaf-or-node and right-or-left.
//
// NB! When stub is used, then (nodeFn) is called as (StubNodeIndex, false, _) as StackCalculator can't produce a node value at index StubNodeIndex.
// NB! The (nodeFn) is not called for the requested leaf itself.
func (v PathBuilder) WalkFor(index uint, nodeFn PathEntryFunc) {
	if v.count <= index {
		panic("illegal value")
	}

	// sibling leaf
	switch sibling := index ^ 1; {
	case sibling < v.count:
		nodeFn(sibling, true, index&1 == 0)
	case v.stubbed:
		nodeFn(StubNodeIndex, false, true)
	}

	for _, level := range v.levels {
		tIndex := siblingIndex(index, level.bitMask)

		isRightSibling := index&level.bitMask == 0

		switch {
		case tIndex < v.count:
			// the tree node was calculated as a part of the normal flow
			nodeFn(tIndex, false, isRightSibling)
			continue
		case isRightSibling:
			lastOfLeftBranch := index | (level.bitMask - 1)
			if lastOfLeftBranch >= v.count-1 {
				// right branch doesn't exist
				if v.stubbed {
					nodeFn(StubNodeIndex, false, true)
				}
				continue
			}
			// right branch is delayed
			tIndex = level.nodeValueR
		default:
			// left branch is delayed
			tIndex = level.nodeValueL
		}

		if tIndex == 0 {
			panic("illegal state")
		}
		nodeFn(tIndex, tIndex == v.count-1 && v.count&1 != 0, isRightSibling)
	}
}

func siblingIndex(index uint, bitMask uint) uint {
	tIndex := index ^ bitMask
	bitMask--
	tIndex |= bitMask
	tIndex += bitMask >> 1
	return tIndex
}
