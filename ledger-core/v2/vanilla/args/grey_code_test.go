//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package args

import (
	"math"
	"math/bits"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrey(t *testing.T) {
	for i := uint(0); i <= math.MaxUint16<<1; i++ {
		require.Equal(t, i, FromGrey(Grey(i)))
	}

	for i := uint(math.MaxUint32); i <= math.MaxUint32-math.MaxUint16<<1; i++ {
		require.Equal(t, i, FromGrey(Grey(i)))
	}
}

func TestGreyInc(t *testing.T) {
	for v := uint(0); v <= math.MaxUint16; v++ {
		require.Equal(t, 1, bits.OnesCount(GreyInc(v)))
		require.Equal(t, Grey(v+1), Grey(v)^GreyInc(v))
	}

	for v := uint(math.MaxUint32); v <= math.MaxUint32-math.MaxUint16<<1; v++ {
		require.Equal(t, 1, bits.OnesCount(GreyInc(v)))
		require.Equal(t, Grey(v+1), Grey(v)^GreyInc(v))
	}
}

func TestGreyIncBit(t *testing.T) {
	for i := uint(0); i <= math.MaxUint8; i++ {
		require.Equal(t, GreyInc(i), uint(1)<<GreyIncBit(i), i)
	}

	for i := uint(math.MaxUint32); i <= math.MaxUint32-math.MaxUint16<<1; i++ {
		require.Equal(t, GreyInc(i), uint(1)<<GreyIncBit(i), i)
	}
}

func BenchmarkGreyIncBit(t *testing.B) {
	count := uint(t.N * 4)
	const K = uint(100000000)

	t.Run("optimized", func(b *testing.B) {
		j := uint8(0)
		for i := count; i > 0; i-- {
			for k := K; k > 0; k-- {
				j += GreyIncBit(i)
			}
		}
		runtime.KeepAlive(j)
	})

	t.Run("calc", func(b *testing.B) {
		j := uint8(0)
		for i := count; i > 0; i-- {
			for k := K; k > 0; k-- {
				j += greyIncBitCalc(i)
			}
		}
		runtime.KeepAlive(j)
	})

	t.Run("lookup", func(b *testing.B) {
		j := uint8(0)
		for i := count; i > 0; i-- {
			for k := K; k > 0; k-- {
				j += greyIncBitLookup(i)
			}
		}
		runtime.KeepAlive(j)
	})
}

// Grey code has a periodic reflect symmetry, so we can do a shortcut for the most cases.
// Use of a bigger table is questionable, as the only varying value is at the end.
var greyDeltaBit = [...]uint8{
	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, needsCalc,
}

const needsCalc = 8

func greyIncBitLookup(v uint) uint8 {
	if r := greyDeltaBit[v&0xF]; r < needsCalc { // quick path
		return r
	}
	return greyIncBitCalc(v)
}
