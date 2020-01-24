//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package args

import "math/bits"

// Converts a linear binary number to a binary-reflected grey code
func Grey(v uint) uint {
	return v ^ (v >> 1)
}

// Converts a binary-reflected grey code to a linear binary number
func FromGrey(g uint) uint {
	g = g ^ (g >> 32) // it is only needed when uint=uint64, so lets hope for the compiler to remove it
	g = g ^ (g >> 16)
	g = g ^ (g >> 8)
	g = g ^ (g >> 4)
	g = g ^ (g >> 2)
	g = g ^ (g >> 1)
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

// Grey code has a periodic reflect symmetry, so we can do a shortcut for the most cases.
// Use of a bigger table is questionable, as the only varying value is at the end.
var greyDeltaBit = [16]uint8{
	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, needsCalc,

	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 5,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 6,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 5,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 7,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 5,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 6,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 5,
	//0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, 4,	0, 1, 0, 2, 0, 1, 0, 3, 0, 1, 0, 2, 0, 1, 0, needsCalc,
}

const needsCalc = 8

// Returns a bit (offset) that will change in grey-code equivalent of v on incrementing it
// The following is always true: 1<<GreyIncBit(v) == GreyInc(v)
func GreyIncBit(v uint) uint8 {
	r := greyDeltaBit[v&0xF]
	if r < needsCalc { // quick path
		return r
	}
	return uint8(bits.Len(GreyInc(v)) - 1)
}
