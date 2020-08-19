// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build extendedtest

package jet

import (
	"bytes"
	"fmt"
	"math"
	"math/bits"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrefixTree_SerializeCombinations(t *testing.T) {
	t.Run("raw   only", func(t *testing.T) {
		t.Parallel()
		ser := PrefixTreeSerializer{}
		pt := PrefixTree{}
		pt.Init()
		m := generateTree(t, &pt, 0, 0, func(t *testing.T, pt *PrefixTree) int {
			return checkTree(t, pt, ser)
		})
		t.Log("Max size: ", m)
		require.Less(t, m, maxUncompressedSize)
		require.Less(t, m, 6700)
	})

	t.Run("raw or lzw", func(t *testing.T) {
		t.Parallel()
		ser := PrefixTreeSerializer{UseLZW: true}
		pt := PrefixTree{}
		pt.Init()
		m := generateTree(t, &pt, 0, 0, func(t *testing.T, pt *PrefixTree) int {
			return checkTree(t, pt, ser)
		})
		t.Log("Max size: ", m)
		require.Less(t, m, maxUncompressedSize)
		require.Less(t, m, 1500)
	})

	t.Run("enum tree", func(t *testing.T) {
		t.Parallel()
		pt := PrefixTree{}
		pt.Init()
		_ = generateTree(t, &pt, 0, 0, func(t *testing.T, pt *PrefixTree) int {
			count := 0
			pt.Enum(func(prefix Prefix, depth uint8) bool {
				count++
				return false
			})
			jetCount := pt.Count()
			if count != jetCount {
				require.Equal(t, jetCount, count)
			}
			return 0
		})
	})
}

func max(m int, i int) int {
	if m < i {
		return i
	}
	return m
}

func generateTree(t *testing.T, pt *PrefixTree, prefix Prefix, baseDepth uint8, handleFn func(t *testing.T, pt *PrefixTree) int) int {
	maxSize := handleFn(t, pt) // zero level

	const maxDepth = 16
	const minDepth = 16

	for depth := baseDepth; depth < maxDepth; depth++ {
		pt.Split(prefix, depth)

		maxSize = max(maxSize, handleFn(t, pt))
		hadSplit := false

		for i := depth + 1; i < maxDepth; i++ {
			pt.Split(prefix, i)
			if i < minDepth {
				maxSize = max(maxSize, generateTree(t, pt, prefix|Prefix(1)<<i, i+1, handleFn))
			} else {
				hadSplit = true
			}
		}
		prefix |= Prefix(1) << depth

		if !hadSplit {
			continue
		}
		maxSize = max(maxSize, handleFn(t, pt))
	}

	return maxSize
}

func checkTree(t *testing.T, pt *PrefixTree, ser PrefixTreeSerializer) int {
	jetCount := pt.Count()

	serBytes := PrefixTreeSerializer{}.SerializeToRawBytes(pt)
	sz := len(serBytes)
	packedSz := sz

	if ser.UseLZW {
		buf := bytes.Buffer{}
		buf.Grow(len(serBytes))
		if err := ser.postSerialize(serBytes, &buf); err != nil {
			require.NoError(t, err) // test performance tweak
		}

		bufBytes := buf.Bytes()
		if bufBytes[0] != RawSerializeV1 {
			packedSz = len(bufBytes)
			serBytes = bufBytes
		}
		// t.Logf("%5d	%2d	%2d	%5d	%2.2f	%5d 	%2d%% \n",
		// 	jetCount, pt.MinDepth(), pt.MaxDepth(),
		// 	sz, float32(sz<<3)/float32(jetCount),
		// 	packedSz,
		// 	packedSz*100/sz,
		// )
	} else {
		// t.Logf("%5d	%2d	%2d	%5d	%2.2f \n",
		// 	jetCount, pt.MinDepth(), pt.MaxDepth(),
		// 	sz, float32(sz<<3)/float32(jetCount),
		// )
		runtime.KeepAlive(jetCount)
	}
	// t.Log(hex.Dump(buf.Bytes()))

	checkSerialized(t, pt, bytes.NewBuffer(serBytes))

	return packedSz
}

func checkSerialized(t *testing.T, pt *PrefixTree, buf *bytes.Buffer) {
	pt2 := PrefixTree{}
	err := PrefixTreeDeserializer{}.DeserializeTo(&pt2, buf)
	if err != nil {
		require.NoError(t, err) // test performance tweak
	}
	if *pt != pt2 {
		require.Equal(t, *pt, pt2) // test performance tweak
	}
}

func TestPrefixTree_Propagate_Get_Performance(t *testing.T) {
	timings := [2]int64{}
	for i := 0; i <= 1; i++ {
		idx := i
		t.Run(fmt.Sprintf("tree=zero16 propagate=%v", idx != 0), func(t *testing.T) {
			pt := NewPrefixTree(idx != 0)
			splitZero(&pt, 0, 15)
			startedAt := time.Now()
			for j := 0; j < 10000000; j++ {
				pt.GetPrefix(math.MaxUint16)
			}
			timings[idx] = int64(time.Since(startedAt))
		})
	}
	require.Less(t, timings[1], timings[0]>>2) // must be at least 4 times faster
}

func TestPrefixTree_Propagate_Get_ZeroThenOne(t *testing.T) {
	for i := 0; i <= 1; i++ {
		pt := NewPrefixTree(i != 0)
		for i := Prefix(0); i <= math.MaxUint16*2; i++ {
			_, l := pt.GetPrefix(i)
			require.Equal(t, uint8(0), l)
		}
		splitZero(&pt, 0, 15)
		mask := Prefix(math.MaxUint16)

		t.Run(fmt.Sprintf("tree=zero16 propagate=%v", pt.autoPropagate), func(t *testing.T) {
			for i := Prefix(0); i <= math.MaxUint16*2; i++ {
				masked := i & mask
				expected := uint8(16)
				if masked != 0 {
					expected = uint8(bits.TrailingZeros(uint(masked)) + 1)
				}
				_, l := pt.GetPrefix(i)
				require.Equal(t, expected, l, i)
			}
		})

		splitOne(&pt, 1, 15)

		t.Run(fmt.Sprintf("tree=zero16+one16 propagate=%v", pt.autoPropagate), func(t *testing.T) {
			for i := Prefix(0); i <= math.MaxUint16*2; i++ {
				masked := i & mask
				expected := uint8(16)
				switch {
				case masked == 0:
				case masked <= 2:
					expected = 2
				case masked == math.MaxUint16:
					expected = 16
				case masked&1 == 0:
					expected = uint8(bits.TrailingZeros(uint(masked)) + 1)
				default:
					expected = uint8(bits.TrailingZeros(^uint(masked)) + 1)
				}
				_, l := pt.GetPrefix(i)
				require.Equal(t, expected, l, i)
			}
		})
	}
}

func TestPrefixTree_Propagate_Get_OneThenZero(t *testing.T) {
	for i := 0; i <= 1; i++ {
		pt := NewPrefixTree(i != 0)
		for i := Prefix(0); i <= math.MaxUint16*2; i++ {
			_, l := pt.GetPrefix(i)
			require.Equal(t, uint8(0), l)
		}
		splitOne(&pt, 0, 15)
		mask := Prefix(math.MaxUint16)

		t.Run(fmt.Sprintf("tree=one16 propagate=%v", pt.autoPropagate), func(t *testing.T) {
			for i := Prefix(0); i <= math.MaxUint16*2; i++ {
				masked := i & mask
				expected := uint8(0)
				switch {
				case masked == 0:
					expected = 1
				case masked == math.MaxUint16:
					expected = 16
				default:
					expected = uint8(bits.TrailingZeros(^uint(masked)) + 1)
				}
				_, l := pt.GetPrefix(i)
				require.Equal(t, expected, l, i)
			}
		})

		splitZero(&pt, 1, 15)

		t.Run(fmt.Sprintf("tree=one16+zero16 propagate=%v", pt.autoPropagate), func(t *testing.T) {
			for i := Prefix(0); i <= math.MaxUint16*2; i++ {
				masked := i & mask
				expected := uint8(16)
				switch {
				case masked == 0:
				case masked <= 2:
					expected = 2
				case masked == math.MaxUint16:
					expected = 16
				case masked&1 == 0:
					expected = uint8(bits.TrailingZeros(uint(masked)) + 1)
				default:
					expected = uint8(bits.TrailingZeros(^uint(masked)) + 1)
				}
				_, l := pt.GetPrefix(i)
				require.Equal(t, expected, l, i)
			}
		})
	}
}
