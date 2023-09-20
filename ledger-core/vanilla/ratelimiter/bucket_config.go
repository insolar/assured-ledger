package ratelimiter

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type BucketConfig struct {
	RefillAmount uint32
	Quantum      uint32
	MaxAmount    uint32
}

func (v BucketConfig) IsZero() bool {
	return v.Quantum == 0
}

func (v BucketConfig) Check() {
	switch {
	case v.Quantum == 0:
		panic(throw.IllegalValue())
	case v.MaxAmount < v.Quantum:
		panic(throw.IllegalValue())
	case v.MaxAmount < v.RefillAmount:
		panic(throw.IllegalValue())
	}
}

func scaleUint32(v uint32, scale float32) uint32 {
	switch {
	case v == 0:
		return 0
	case scale <= 0:
		return 0
	default:
		f := float32(v) * scale
		if f > math.MaxUint32 {
			return math.MaxUint32
		}
		v = uint32(f)
		if v > 0 {
			return v
		}
		return 1
	}
}

func (v BucketConfig) Scale(refillRatio, maxRatio float32) BucketConfig {
	v.RefillAmount = scaleUint32(v.RefillAmount, refillRatio)
	v.MaxAmount = scaleUint32(v.MaxAmount, maxRatio)
	return v
}
