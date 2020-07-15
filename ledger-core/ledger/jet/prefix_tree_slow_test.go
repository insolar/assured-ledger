// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package jet

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrefixTree_SerializeCombinations(t *testing.T) {
	t.Run("raw   only", func(t *testing.T) {
		t.Skip("too slow")
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
		t.Skip("too slow")
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
