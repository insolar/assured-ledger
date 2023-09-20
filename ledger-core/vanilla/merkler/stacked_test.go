package merkler

import (
	"fmt"
	"math"
	"math/bits"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func TestStackedCalculator_Unbalanced_AllEntries(t *testing.T) {
	md := NewForkingCalculator(xorPairDigester{}, cryptkit.Digest{})

	for bit := uint64(1); bit != 0; bit <<= 1 {
		md.AddNext(newBits64(bit))
		require.Equal(t, bit<<1-1, md.ForkSequence().FinishSequence().FoldToUint64())
	}
	require.Equal(t, 64, md.Count())
	require.Equal(t, uint64(math.MaxUint64), md.ForkSequence().FinishSequence().FoldToUint64())

	md2 := md.ForkSequence()
	for bit := uint64(1) << 63; bit != 0; bit >>= 1 {
		md2.AddNext(newBits64(bit))
		require.Equal(t, bit-1, md2.ForkSequence().FinishSequence().FoldToUint64())
	}
	require.Equal(t, uint64(0), md2.FinishSequence().FoldToUint64())

	for bit := uint64(1); bit != 0; bit <<= 1 {
		md.AddNext(newBits64(bit))
		require.Equal(t, ^(bit<<1 - 1), md.ForkSequence().FinishSequence().FoldToUint64())
	}
	require.Equal(t, 128, md.Count())
	require.Equal(t, uint64(0), md.FinishSequence().FoldToUint64())
}

func TestStackedCalculator_EntryPos(t *testing.T) {
	// this table is a number of right-to-left transitions
	expectedPos := [0x3b]byte{
		//  0  1  2  3  4  5  6  7  8  9  A  B  C  D  E  F
		0, 1, 1, 2, 1, 2, 2, 3, 1, 2, 2, 3, 2, 3, 3, 4,
		1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
		1, 2, 2, 3, 2, 3, 3, 4, 2, 3, 3, 4, 3, 4, 4, 5,
		2, 3, 3, 4, 3, 4, 4, 5, 3, 4, 4, // overflow
	}

	for balanced := 0; balanced <= 1; balanced++ {
		t.Run(fmt.Sprintf("balanced=%v", balanced != 0), func(t *testing.T) {
			for markBit := uint64(1); markBit < uint64(1)<<uint8(len(expectedPos)); /* to avoid overflow in this test */ markBit <<= 1 {
				var md ForkingCalculator
				if balanced != 0 {
					md = NewForkingCalculator(xorShiftPairDigester{}, cryptkit.NewDigest(newBits64(0), "uint64"))
				} else {
					md = NewForkingCalculator(xorShiftPairDigester{}, cryptkit.Digest{})
				}

				for bit := uint64(1); bit != markBit; bit <<= 1 {
					md.AddNext(newBits64(0))
				}
				md.AddNext(newBits64(markBit))

				markBitPos := bits.Len64(markBit) - 1
				expected := byte(markBitPos) + expectedPos[markBitPos]

				v := md.ForkSequence().FinishSequence().FoldToUint64()
				require.Equal(t, 1, bits.OnesCount64(v), "0x%x", markBitPos)
				require.Equal(t, expected, byte(bits.Len64(v)-1), "0x%x", markBitPos)

				for bit := uint64(markBit) << 1; bit != 0; bit <<= 1 {
					md.AddNext(newBits64(0))
					v := md.ForkSequence().FinishSequence().FoldToUint64()
					require.Equal(t, expected, byte(bits.Len64(v)-1), "0x%x", markBitPos)
				}
				require.Equal(t, 64, md.Count())
			}
		})
	}
}

func TestStackedCalculator_Balanced_AllEntries(t *testing.T) {
	const unbalanced = uint64(1 << 56)
	expectedUnbalanced := [32]byte{
		1, 0, 1, 0, 2, 1, 1, 0, 3, 2, 2, 1, 2, 1, 1, 0,
		4, 3, 3, 2, 3, 2, 2, 1, 3, 2, 2, 1, 2, 1, 1, 0}

	md := NewForkingCalculator(xorCountPairDigester{}, cryptkit.NewDigest(newBits64(unbalanced), "uint64"))

	require.Equal(t, unbalanced, md.ForkSequence().FinishSequence().FoldToUint64())

	for bit := uint64(1); bit <= 1<<31; bit <<= 1 {
		md.AddNext(newBits64(bit))
		v := md.ForkSequence().FinishSequence().FoldToUint64()

		require.Equal(t, uint64(expectedUnbalanced[bits.Len64(bit)-1]), v>>56, bits.Len64(bit))
		require.Equal(t, bit<<1-1, v&math.MaxUint32, bits.Len64(bit))
	}
	require.Equal(t, 32, md.Count())
	require.Equal(t, uint64(math.MaxUint32), md.ForkSequence().FinishSequence().FoldToUint64())

	md2 := md.ForkSequence()
	for bit := uint64(1) << 31; bit != 0; bit >>= 1 {
		md2.AddNext(newBits64(bit))
		v := md2.ForkSequence().FinishSequence().FoldToUint64()
		//fmt.Println(v>>56)
		require.Equal(t, bit-1, v&math.MaxUint32)
	}
	require.Equal(t, uint64(0), md2.FinishSequence().FoldToUint64())

	for bit := uint64(1); bit <= 1<<31; bit <<= 1 {
		md.AddNext(newBits64(bit))
		v := md.ForkSequence().FinishSequence().FoldToUint64()
		//fmt.Println(v>>56)
		require.Equal(t, ^uint32(bit<<1-1), uint32(v))
	}
	require.Equal(t, 64, md.Count())
	require.Equal(t, uint64(0), md.FinishSequence().FoldToUint64())
}

func newBits64(v uint64) *longbits.Bits64 {
	v64 := longbits.NewBits64(v)
	return &v64
}

type xorPairDigester struct{}

func (p xorPairDigester) GetDigestSize() int {
	return 8
}

func (p xorPairDigester) DigestPair(digest0 longbits.FoldableReader, digest1 longbits.FoldableReader) cryptkit.Digest {
	return cryptkit.NewDigest(newBits64(digest0.FoldToUint64()^digest1.FoldToUint64()), "uint64")
}

func (p xorPairDigester) GetDigestMethod() cryptkit.DigestMethod {
	return "xor64"
}

type xorCountPairDigester struct {
	xorPairDigester
}

func (p xorCountPairDigester) DigestPair(digest0 longbits.FoldableReader, digest1 longbits.FoldableReader) cryptkit.Digest {
	const topByteMask = ^uint64(math.MaxUint64 >> 8)

	v0 := digest0.FoldToUint64()
	v1 := digest1.FoldToUint64()
	xored := (v0 ^ v1) &^ topByteMask
	//	counted := uint64(0)
	counted := v0&topByteMask + v1&topByteMask
	return cryptkit.NewDigest(newBits64(counted|xored), "uint64")
}

type xorShiftPairDigester struct {
	xorPairDigester
}

func (p xorShiftPairDigester) DigestPair(digest0 longbits.FoldableReader, digest1 longbits.FoldableReader) cryptkit.Digest {
	v0 := digest0.FoldToUint64()
	v1 := digest1.FoldToUint64()
	return cryptkit.NewDigest(newBits64(v0^(v1<<1)), "uint64")
}
