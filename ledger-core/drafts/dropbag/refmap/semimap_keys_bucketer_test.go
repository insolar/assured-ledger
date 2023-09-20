package refmap

import (
	"fmt"
	"math"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func TestBucketing(t *testing.T) {
	m := NewRefLocatorMap()
	m.keys.SetHashSeed(0)

	const keyCount = int(1e2)
	for i := keyCount; i > 0; i-- {
		refLocal := makeLocal(i)
		for j := i & 3; j >= 0; j-- {
			refBase := makeLocal(i + j*1e6)
			ref := reference.NewNoCopy(&refBase, &refLocal)
			locator := ValueLocator(i*100 + 99)
			m.Put(ref, locator)
			// fmt.Println("	REF[", i, "]	", refLocal.GetPulseNumber(), refBase.GetPulseNumber(), "=>", locator)
		}
	}

	wb := m.FillLocatorBuckets(WriteBucketerConfig{
		ExpectedPerBucket: 100,
		UsePrimes:         false,
	})

	fmt.Println("Params", wb.AdjustedBucketSize(), wb.BucketCount())

	if len(wb.overflowEntries) > 0 {
		minOverflow := math.MaxInt32
		maxOverflow := 0
		for _, v := range wb.overflowEntries {
			n := len(v)
			if n < minOverflow {
				minOverflow = n
			}
			if n > maxOverflow {
				maxOverflow = n
			}
		}

		fmt.Println("Overflown", len(wb.overflowEntries), minOverflow, maxOverflow)
	}

	_ = writeLocatorBuckets(&wb, math.MaxUint32)
}
