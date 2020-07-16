// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jetalloc

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const SplitMedian = 7 // makes 0 vs 1 ratio like [0..6] vs [7..15]
// this enables left branches of jets to be ~23% less loaded

//
// Default prefix calculator, requires 12 bytes for 16 bit prefix and uses SplitMedian const for mis-balancing.
//
// Recommended use:
// 		bitPrefix := NewPrefixCalc().FromXXX(prefixTree.MaxDepth(), reference)
//        or
// 		bitPrefix := NewPrefixCalc().FromXXX(16, reference)
//		...
//		bitPrefix, bitPrefixLen = prefixTree.GetPrefix(bitPrefix)
//
func NewPrefixCalc() PrefixCalc {
	return PrefixCalc{4, SplitMedian}
}

//
// Converts a byte sequence into a bit prefix for PrefixTree.
//
// Must set OverlapOfs>0 when a structured header is present within a byte sequence.
// When OverlapOfs !=0, then the calculator will mix b[n]^b[n + OverlapOfs]
//
type PrefixCalc struct {
	OverlapOfs  uint8
	SplitMedian uint8
}

// Converts data[:OverlapOfs + (prefixLen)/2] into prefixLen bits.
func (p PrefixCalc) FromSlice(prefixLen int, data []byte) jet.Prefix {
	switch {
	case prefixLen < 0 || prefixLen > 32:
		panic(throw.IllegalValue())
	case prefixLen == 0:
		return 0
	}

	return p.fromSlice(prefixLen, data)
}

// Converts data[:OverlapOfs + (prefixLen)/2] into prefixLen bits.
func (p PrefixCalc) FromReader(prefixLen int, data io.Reader) (jet.Prefix, error) {
	switch {
	case prefixLen < 0 || prefixLen > 32:
		panic(throw.IllegalValue())
	case data == nil:
		panic(throw.IllegalValue())
	case prefixLen == 0:
		return 0, nil
	}

	dataBuf := make([]byte, int(p.OverlapOfs)+(prefixLen+1)>>1)
	switch n, err := data.Read(dataBuf); {
	case err != nil:
		return 0, err
	case n != len(dataBuf):
		return 0, throw.FailHere("insufficient data length")
	}

	return p.fromSlice(prefixLen, dataBuf), nil
}

func (p PrefixCalc) fromSlice(prefixLen int, data []byte) jet.Prefix {
	result := jet.Prefix(0)
	bit := jet.Prefix(1)

	for i, d := range data {
		if p.OverlapOfs > 0 {
			// TODO check quality of distribution because of XOR
			d ^= data[i+int(p.OverlapOfs)]
		}

		if d&0xF >= p.SplitMedian {
			result |= bit
		}

		if prefixLen == 1 {
			// odd length
			return result
		}
		bit <<= 1

		if (d >> 4) >= p.SplitMedian {
			result |= bit
		}
		prefixLen -= 2
		if prefixLen == 0 {
			return result
		}
		bit <<= 1
	}

	panic(throw.FailHere("insufficient data length"))
}


