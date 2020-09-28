// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"math"
	"math/bits"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/args"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func BucketConfigByThroughput(bytePerS int, refillUnit uint64, burst int, quantum uint32) BucketConfig {
	if bytePerS <= 0 {
		return BucketConfig{
			RefillAmount: math.MaxUint32,
			Quantum:      quantum,
			MaxAmount:    math.MaxUint32,
		}
	}

	refillAmount := uint64(bytePerS) / refillUnit
	if refillAmount == 0 {
		refillAmount = 1
	}
	if burst <= 0 {
		burst = 1
	}
	maxAmount := (refillAmount*uint64(burst) + uint64(quantum) - 1) / uint64(quantum)
	maxAmount *= uint64(quantum)
	if maxAmount > math.MaxUint32 {
		maxAmount = math.MaxUint32
	}

	return BucketConfig{
		RefillAmount: uint32(refillAmount),
		Quantum:      quantum,
		MaxAmount:    uint32(maxAmount),
	}
}

func ThroughputQuantizer(quantum uint32, samplesPerSecond uint64, limits ...int) (increment uint32, refillUnit uint64, scaleUnit uint32) {
	switch max := maxLimit(0, limits...); {
	case max == 0:
		return 0, 1, 1 // no limits
	case max > math.MaxUint32:
		scaleUnit = 1 << bits.Len64(uint64(max)/math.MaxUint32)
	default:
		scaleUnit = 1
	}

	min := minLimit(math.MaxInt64, int64(quantum), limits...)
	switch {
	case min == math.MaxInt64:
		return 0, 1, 1 // no limits
	case min <= 0:
		panic(throw.Impossible())
	case len(limits) > 1:
		if gcd := uint64(args.GCDListInt(int(samplesPerSecond)<<2, limits[0], limits[1:]...)); gcd >= samplesPerSecond {
			min = int64(gcd)
		}
	}

	minSamples := uint64(min) / uint64(scaleUnit)
	minSamples /= samplesPerSecond

	bitExtra := 0
	if minSamples > 1 {
		bitExtra = bits.Len64(minSamples - 1)
	}
	if bitExtra < 3 {
		return 1, samplesPerSecond * uint64(scaleUnit), scaleUnit
	}
	bitExtra -= 2
	if bitExtra > 8 {
		bitExtra = 8
	}
	return 1 << bitExtra, (samplesPerSecond << bitExtra) * uint64(scaleUnit), scaleUnit
}

func minLimit(v, lowCut int64, vv ...int) int64 {
	for i := range vv {
		vvv := int64(vv[i])
		switch {
		case vvv <= 0:
			continue
		case vvv < lowCut:
			vvv = lowCut
		}
		if v > vvv {
			v = vvv
		}
	}
	return v
}

func maxLimit(v int64, vv ...int) int64 {
	for i := range vv {
		vvv := int64(vv[i])
		switch {
		case vvv <= 0:
			continue
		case vvv > v:
			v = vvv
		}
	}
	return v
}

func quantizeCeiling(v, quantum uint32) uint32 {
	if v < quantum {
		return quantum
	}
	a := (uint64(v) + uint64(quantum) - 1) / uint64(quantum)
	a *= uint64(quantum)
	if a > math.MaxUint32 {
		return uint32(a - uint64(quantum))
	}
	return uint32(a)
}

func quantizeFloor(v, quantum uint32) uint32 {
	if v < quantum {
		return 0
	}
	a := uint64(v) / uint64(quantum)
	return uint32(a * uint64(quantum))
}
