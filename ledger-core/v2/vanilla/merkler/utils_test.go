// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package merkler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnbalancedBitCount(t *testing.T) {
	for i := uint8(0); i <= 64; i++ {
		assert.Equal(t, uint8(0), UnbalancedBitCount(uint(1)<<i-1, i))
	}

	for bitCount := uint8(1); bitCount <= 8; bitCount++ {
		for n, i := bitCount, uint(0); n > 0; n-- {
			for j := uint(1) << (n - 1); j > 0; i, j = i+1, j-1 {
				v := UnbalancedBitCount(i, bitCount)
				//t.Logf("%2d %3d %2d", bitCount, i, v)
				require.Equal(t, n, v, "bits=%d index=%d", bitCount, i)
			}
		}
	}
}
