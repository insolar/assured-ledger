// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"context"
	"math"
	"math/bits"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/capacity"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/ratelimiter"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type TrafficConfig struct {
	TotalRead, TotalWrite int
	PeerRead, PeerWrite   int

	SamplingPeriod time.Duration
	BurstPeriods   int
}

func NewTrafficController(cfg TrafficConfig) *TrafficController {
	if cfg.SamplingPeriod == 0 {
		cfg.SamplingPeriod = 500 * time.Microsecond
	}
	sps := time.Second / cfg.SamplingPeriod
	if sps <= 0 {
		panic(throw.IllegalValue())
	}

	quantum := uint32(1024)
	increment, refillUnit, scaleUnit := quantizeCfg(quantum, uint64(sps),
		cfg.TotalRead, cfg.TotalWrite, cfg.PeerRead, cfg.PeerWrite)

	tc := &TrafficController{
		ctl:          ratelimiter.NewController(scaleUnit, cfg.SamplingPeriod),
		maxIncrement: increment,
	}
	tc.ctl.Init()

	if increment == 0 {
		return tc
	}

	if cfg.TotalRead > 0 || cfg.TotalWrite > 0 {
		tc.total = ratelimiter.NewRWBucket(tc.ctl.Root(),
			bucketConfig(cfg.TotalRead, refillUnit, cfg.BurstPeriods, quantum),
			bucketConfig(cfg.TotalWrite, refillUnit, cfg.BurstPeriods, quantum),
		)
	}

	if cfg.PeerRead > 0 || cfg.PeerWrite > 0 {
		tc.read = bucketConfig(cfg.PeerRead, refillUnit, cfg.BurstPeriods, quantum)
		tc.write = bucketConfig(cfg.PeerWrite, refillUnit, cfg.BurstPeriods, quantum)
	}

	return tc
}

func bucketConfig(bytePerS int, refillUnit uint64, burst int, quantum uint32) ratelimiter.BucketConfig {
	if bytePerS <= 0 {
		return ratelimiter.BucketConfig{
			RefillAmount: math.MaxUint32,
			Quantum:      quantum,
			MaxAmount:    math.MaxUint32,
		}
	}

	refillAmount := uint64(bytePerS) / refillUnit
	if refillAmount <= 0 {
		refillAmount = 1
	}
	if burst <= 0 {
		burst = 1
	}
	maxAmount := uint32((refillAmount*uint64(burst) + uint64(quantum) - 1) / uint64(quantum))
	maxAmount *= quantum

	return ratelimiter.BucketConfig{
		RefillAmount: uint32(refillAmount),
		Quantum:      quantum,
		MaxAmount:    maxAmount,
	}
}

func quantizeCfg(quantum uint32, samplesPerSecond uint64, limits ...int) (increment uint32, refillUnit uint64, scaleUnit uint32) {
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
	}

	minSamples := uint64(min)
	minSamples /= uint64(scaleUnit)
	minSamples /= samplesPerSecond

	bitExtra := bits.Len64(minSamples)
	if bitExtra < 2 {
		panic(throw.IllegalValue())
	}
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

type TrafficController struct {
	ctl          ratelimiter.Controller
	levels       capacity.Uint8Levels
	total        *ratelimiter.RWBucket
	maxIncrement uint32

	read, write ratelimiter.BucketConfig
	//max
}

func (p *TrafficController) Start(ctx context.Context) {
	if p.maxIncrement > 0 {
		p.ctl.Start(ctx)
	}
}

func (p *TrafficController) SetThroughput(percent uint8) {
	if percent > 100 {
		percent = 100
	}
	p.ctl.SetIncrement(uint32(percent) * p.maxIncrement / 100)
}

func (p *TrafficController) GetThroughput() uint8 {
	if p.maxIncrement == 0 {
		return 100
	}
	return uint8(p.ctl.GetIncrement() * 100 / p.maxIncrement)
}

func (p *TrafficController) SetLevel(level capacity.Level) {
	p.SetThroughput(level.ChooseUint8(p.levels))
}

func (p *TrafficController) NewRWBucket() ratelimiter.RWRateQuota {
	switch {
	case p.maxIncrement == 0:
		return nil
	case p.total == nil:
		return ratelimiter.NewRWBucket(p.ctl.Root(), p.read, p.write)
	case p.read.IsZero() && p.write.IsZero():
		return p.total
	default:
		return ratelimiter.NewRWChildBucket(p.total, p.read, p.write)
	}
}

//func (p *TrafficController) NewReadBucket() ratelimiter.RateQuota {
//	switch {
//	case p.maxIncrement == 0:
//		return nil
//	case p.total == nil:
//		return ratelimiter.NewBucket(p.ctl.Root(), p.read)
//	case p.read.IsZero() && p.write.IsZero():
//		return p.total
//	default:
//		return ratelimiter.NewChildBucket(p.total, p.read)
//	}
//}
