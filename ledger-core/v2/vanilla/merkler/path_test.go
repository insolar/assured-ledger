package merkler

import (
	"fmt"
	"math/bits"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/args"
)

func TestPathBuilder_WalkFor(t *testing.T) {
	pb := NewPathBuilder(32, false)
	for i := uint(0); i < pb.count; i++ {
		fmt.Println("Index: ", i, " =============================== ")
		pb.WalkFor(i, func(prefix uint, level uint8) {
			idx := pb.prefixToIndex(prefix, level)
			fmt.Printf("	L%d=%0*b		%d\n", level, pb.depth-level, prefix, idx)
		})
	}
}

func TestPrintMerkleTrace(t *testing.T) {
	node := 'A'
	idx := 0
	const maxLevel = 4
	for i := uint(0); i < 1<<(1+maxLevel)-1; i++ {
		fmt.Printf("%2d V =%2d\n", idx, i)
		idx++
		fmt.Printf("%2d ", idx)
		idx++

		level := args.GreyIncBit(i + 1)
		if level > 0 && bits.Len(i) <= int(level) || level == 0 && i >= 1<<maxLevel {
			fmt.Println()
			continue
		}
		prefix := i >> (level + 1) // TODO this is wrong
		//if prefix >= 8 {
		//	fmt.Println()
		//	continue
		//}
		fmt.Printf("L%d=%0*b	(%c)\n", level, maxLevel-level-1, prefix, node)
		node++
	}
}
