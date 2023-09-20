package merkler

import (
	"fmt"
	"math"
	"math/bits"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type item = uint64

func buildTrace(t *testing.T, count uint, unbalancedStub cryptkit.Digest, skipEmpty bool) (result []item, sum uint64) {
	result = make([]item, 0, int(count)<<1+bits.Len(count))

	md := NewStackedCalculator(xorPairDigester{}, unbalancedStub, func(item longbits.FoldableReader, _ bool) {
		switch {
		case item != nil:
			u := item.FoldToUint64()
			if count <= 64 {
				require.NotEqual(t, uint64(0), u)
			}
			result = append(result, u)
		case skipEmpty:
			return
		default:
			result = append(result, 0)
		}
	})

	for i := uint(0); i < count; i++ {
		bit := bits.RotateLeft64(1, int(i))
		md.AddNext(newBits64(bit))
	}

	sum = md.FinishSequence().FoldToUint64()

	var mask uint64

	require.Equal(t, int(count), md.Count())
	count &= 0x7F
	switch {
	case count < 64:
		mask = ^uint64(math.MaxUint64 << count)
	case count > 64:
		mask = math.MaxUint64 << (count - 64)
	default:
		mask = ^uint64(0)
	}
	if unbalancedStub.IsEmpty() {
		require.Equal(t, mask, sum)
	}

	return
}

//nolint
func printTrace(trace []item, count uint) {
	for i, item := range trace {
		switch index := uint(i >> 1); {
		case index >= count:
			index = uint(i) - count
			fmt.Printf("%3d# ", index)
		case i&1 == 1:
			fmt.Printf("%3d* ", index)
		default:
			fmt.Printf("%3d  ", index)
		}
		if item == 0 {
			fmt.Println()
			continue
		}
		fmt.Printf("%064b\n", item)
	}
}

func TestPathBuilder_WalkUnstubbed(t *testing.T) {
	for count := uint(0); count < 64; count++ {
		leafCount := 1 + count
		t.Run(fmt.Sprint(leafCount), func(t *testing.T) {
			testWalk(t, leafCount, cryptkit.Digest{})
		})
	}
}

func TestPathBuilder_WalkStubbed(t *testing.T) {
	stubDigest := cryptkit.NewDigest(newBits64(uint64(1)<<63), "x")

	for count := uint(0); count < 64; count++ {
		leafCount := 1 + count
		t.Run(fmt.Sprint(leafCount), func(t *testing.T) {
			testWalk(t, leafCount, stubDigest)
		})
	}
}

func testWalk(t *testing.T, leafCount uint, stubDigest cryptkit.Digest) {

	var stub uint64
	if !stubDigest.IsEmpty() {
		stub = stubDigest.FoldToUint64()
	}

	trace, expected := buildTrace(t, leafCount, stubDigest, false)
	// fmt.Println("\nTrace: ", leafCount, " ============================================================== ")
	// printTrace(trace, leafCount)

	pb := NewPathBuilder(leafCount, !stubDigest.IsEmpty())

	// fmt.Println()
	// for i, level := range pb.levels {
	// 	fmt.Printf("L%d	%3d %3d\n", i+1, level.nodeValueL, level.nodeValueR)
	// }
	//
	// fmt.Println()

	for i := uint(0); i < pb.count; i++ {
		// fmt.Println("\nIndex: ", i, "/", leafCount, " ======== ")

		total := uint64(1) << i
		// fmt.Printf("     %064b\n", total)
		pb.WalkFor(i, func(index uint, isLeaf, _ bool) {

			switch {
			case isLeaf:
				// fmt.Printf("%3d  ", index)
				index <<= 1
			case pb.stubbed && index == 0:
				// fmt.Printf("stub %064b\n", stub)
				total ^= stub
				return
			case index < pb.count:
				// fmt.Printf("%3d* ", index)
				index = index<<1 + 1
			default:
				// fmt.Printf("%3d# ", index)
				index = pb.count + index
			}
			if index >= uint(len(trace)) {
				// fmt.Println("-")
				return
			}

			u := trace[index]
			switch {
			case leafCount > 64:
				//
			case u == 0:
				require.Fail(t, "empty entry")
			default:
				v := u &^ stub
				assert.Zero(t, v&total)
			}
			total ^= u
			// fmt.Printf("%064b\n", u)
		})
		// fmt.Printf("ALL  %064b\n", total)
		assert.Equal(t, expected, total)
	}
}
