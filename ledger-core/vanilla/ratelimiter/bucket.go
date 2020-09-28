// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

type RateQuota interface {
	TakeQuota(max int64) int64
	TakeQuotaNoWait(max int64) int64
}

func NewBucket(bm *PeriodManager, cfg BucketConfig) *Bucket {
	cfg.Check()

	return &Bucket{state: BucketState{
		filledAmount: atomickit.NewUint32(cfg.RefillAmount),
		bucketConfig: cfg,
	}, refiller: PeriodRefiller{
		manager:     bm,
		periodCount: atomickit.NewUint64(bm.currentPeriod.Load()),
	}}
}

func NewChildBucket(parent ParentBucket, cfg BucketConfig) *ChildBucket {
	cfg.Check()

	bm := parent.(periodBucket).GetManager()
	b := &ChildBucket{state: BucketState{
		bucketConfig: cfg,
	}, refiller: PeriodRefiller{
		manager:     bm,
		periodCount: atomickit.NewUint64(bm.currentPeriod.Load()),
	}, parent: parent}
	return b
}

func NewChildBucketRatio(parent ParentBucket, refillRatio, maxRatio float32) *ChildBucket {
	return NewChildBucket(parent, parent.CopyConfig().Scale(refillRatio, maxRatio))
}

type ParentBucket interface {
	CopyConfig() BucketConfig
	TakeQuotaNoScale(uint32) uint32
}

type periodBucket interface {
	GetManager() *PeriodManager
}

/******************************************************/

var _ ParentBucket = &Bucket{}
var _ periodBucket = &Bucket{}

type Bucket struct {
	state    BucketState
	refiller PeriodRefiller
}

func (p *Bucket) TakeQuota(max int64) int64 {
	return p.refiller.TakeQuota(max, &p.state, p.refill)
}

func (p *Bucket) TakeQuotaNoWait(max int64) int64 {
	return p.refiller.TakeQuotaNoWait(max, &p.state, p.refill)
}

func (p *Bucket) TakeQuotaNoScale(max uint32) uint32 {
	return p.state.TakeQuotaNoScale(max, p.refill)
}

func (p *Bucket) CopyConfig() BucketConfig {
	return p.state.config()
}

func (p *Bucket) GetManager() *PeriodManager {
	return p.refiller.manager
}

func (p *Bucket) refill(uint32) uint32 {
	x := p.refiller.GetRefillCount()
	return p.state.PeriodsToRefill(x)
}

/******************************************************/

var _ ParentBucket = &ChildBucket{}
var _ periodBucket = &ChildBucket{}

type ChildBucket struct {
	state    BucketState
	refiller PeriodRefiller
	quota    RefillQuota
	parent   ParentBucket
}

func (p *ChildBucket) TakeQuota(max int64) int64 {
	return p.refiller.TakeQuota(max, &p.state, p.refill)
}

func (p *ChildBucket) TakeQuotaNoWait(max int64) int64 {
	return p.refiller.TakeQuotaNoWait(max, &p.state, p.refill)
}

func (p *ChildBucket) TakeQuotaNoScale(max uint32) uint32 {
	return p.state.TakeQuotaNoScale(max, p.refill)
}

func (p *ChildBucket) CopyConfig() BucketConfig {
	return p.state.config()
}

func (p *ChildBucket) GetManager() *PeriodManager {
	return p.refiller.manager
}

func (p *ChildBucket) refill(needed uint32) uint32 {
	x := p.refiller.GetRefillCount()
	return p.quota.Refill(needed, x, &p.state, p.parent.TakeQuotaNoScale)
}
