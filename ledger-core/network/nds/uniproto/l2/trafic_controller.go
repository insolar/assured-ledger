// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l2

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/capacity"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/ratelimiter"
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
	increment, refillUnit, scaleUnit := ratelimiter.ThroughputQuantizer(quantum, uint64(sps),
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
			ratelimiter.BucketConfigByThroughput(cfg.TotalRead, refillUnit, cfg.BurstPeriods, quantum),
			ratelimiter.BucketConfigByThroughput(cfg.TotalWrite, refillUnit, cfg.BurstPeriods, quantum),
		)
	}

	if cfg.PeerRead > 0 || cfg.PeerWrite > 0 {
		tc.read = ratelimiter.BucketConfigByThroughput(cfg.PeerRead, refillUnit, cfg.BurstPeriods, quantum)
		tc.write = ratelimiter.BucketConfigByThroughput(cfg.PeerWrite, refillUnit, cfg.BurstPeriods, quantum)
	}

	return tc
}

type TrafficController struct {
	ctl          *ratelimiter.Controller
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
