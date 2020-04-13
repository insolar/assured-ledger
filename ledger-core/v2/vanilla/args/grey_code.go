// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package args

import "math/bits"

// Converts a linear binary number to a binary-reflected grey code
func Grey(v uint) uint {
	return v ^ (v >> 1)
}

// Converts a binary-reflected grey code to a linear binary number
func FromGrey(g uint) uint {
	if bits.UintSize == 64 {
		g ^= g >> 32
	}
	g ^= g >> 16
	g ^= g >> 8
	g ^= g >> 4
	g ^= g >> 2
	g ^= g >> 1
	return g
}

// Gives a grey-code increment for the given binary. Result always has only one non-zero bit.
// The following is always true: Grey(v) ^ GreyInc(v) == Grey(v + 1)
func GreyInc(v uint) uint {
	// This can also be calculated in a classical way with parity (count non-zero bits) of value, but it will be slower
	//
	// Classical gray_inc(x):
	//   if parity of x is even:
	//	 return x ^ 1
	//   if parity of x is odd:
	//	 y := rightmost 1 bit in x
	//	 return x ^ (y << 1)
	//

	// The fastest way is the shorter version of Grey(v) ^ Grey(v+1)
	return Grey(v ^ (v + 1))
}

// Returns a bit (offset) that will change in grey-code equivalent of v on incrementing it
// The following is always true: 1<<GreyIncBit(v) == GreyInc(v)
func GreyIncBit(v uint) uint8 {
	if v&1 == 0 {
		return 0 // a faster way
	}
	return greyIncBitCalc(v)
}

func greyIncBitCalc(v uint) uint8 {
	return uint8(bits.Len(GreyInc(v)) - 1)
}
