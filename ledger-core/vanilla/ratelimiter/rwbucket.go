// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package ratelimiter

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

type RWRateQuota interface {
	RateQuota
	TakeWriteQuota(max int64) int64
	TakeWriteQuotaNoWait(max int64) int64
	WriteBucket() RateQuota
}

func NewRWBucket(bm *PeriodManager, read, write BucketConfig) *RWBucket {
	read.Check()
	write.Check()

	return &RWBucket{RWSubBucket{
		stateR: BucketState{
			filledAmount: atomickit.NewUint32(read.RefillAmount),
			bucketConfig: read,
		},
		stateW: BucketState{
			filledAmount: atomickit.NewUint32(write.RefillAmount),
			bucketConfig: write,
		},
		refiller: PeriodRefiller{
			manager:     bm,
			periodCount: atomickit.NewUint64(bm.currentPeriod.Load()),
		}}}
}

func NewRWChildBucket(parent ParentRWBucket, read, write BucketConfig) *RWChildBucket {
	read.Check()
	write.Check()

	bm := parent.(periodBucket).GetManager()
	return &RWChildBucket{RWChildSubBucket{
		stateR: BucketState{
			bucketConfig: read,
		}, stateW: BucketState{
			bucketConfig: write,
		}, refiller: PeriodRefiller{
			manager:     bm,
			periodCount: atomickit.NewUint64(bm.currentPeriod.Load()),
		}, parent: parent}}
}

func NewRWChildBucketRatio(parent ParentRWBucket, refillRatio, maxRatio float32) *RWChildBucket {
	cfgR, cfgW := parent.CopyRWConfig()
	return NewRWChildBucket(parent, cfgR.Scale(refillRatio, maxRatio), cfgW.Scale(refillRatio, maxRatio))
}

var _ ParentRWBucket = &RWBucket{}
var _ RWRateQuota = &RWBucket{}
var _ periodBucket = &RWBucket{}

type RWBucket struct {
	s RWSubBucket
}

func (p *RWBucket) GetManager() *PeriodManager {
	return p.s.refiller.manager
}

func (p *RWBucket) CopyRWConfig() (read, write BucketConfig) {
	return p.s.stateR.config(), p.s.stateW.config()
}

func (p *RWBucket) TakeQuota(max int64) int64 {
	return p.s.refiller.TakeQuota(max, &p.s.stateR, p.s.refillR)
}

func (p *RWBucket) TakeQuotaNoWait(max int64) int64 {
	return p.s.refiller.TakeQuotaNoWait(max, &p.s.stateR, p.s.refillR)
}

func (p *RWBucket) TakeWriteQuota(max int64) int64 {
	return p.s.refiller.TakeQuota(max, &p.s.stateW, p.s.refillW)
}

func (p *RWBucket) TakeWriteQuotaNoWait(max int64) int64 {
	return p.s.refiller.TakeQuotaNoWait(max, &p.s.stateW, p.s.refillW)
}

func (p *RWBucket) TakeReadQuotaNoScale(max uint32) uint32 {
	return p.s.stateR.TakeQuotaNoScale(max, p.s.refillR)
}

func (p *RWBucket) TakeWriteQuotaNoScale(max uint32) uint32 {
	return p.s.stateW.TakeQuotaNoScale(max, p.s.refillW)
}

func (p *RWBucket) WriteBucket() RateQuota {
	return &p.s
}

type RWSubBucket struct {
	stateR   BucketState
	stateW   BucketState
	refiller PeriodRefiller
}

func (p *RWSubBucket) TakeQuota(max int64) int64 {
	return p.refiller.TakeQuota(max, &p.stateW, p.refillW)
}

func (p *RWSubBucket) TakeQuotaNoWait(max int64) int64 {
	return p.refiller.TakeQuotaNoWait(max, &p.stateW, p.refillW)
}

func (p *RWSubBucket) refillR(uint32) uint32 {
	x := p.refiller.GetRefillCount()
	p.stateW.ForceRefill(p.stateW.PeriodsToRefill(x))
	return p.stateR.PeriodsToRefill(x)
}

func (p *RWSubBucket) refillW(uint32) uint32 {
	x := p.refiller.GetRefillCount()
	p.stateR.ForceRefill(p.stateR.PeriodsToRefill(x))
	return p.stateW.PeriodsToRefill(x)
}

/**********************************************************/

var _ ParentRWBucket = &RWChildBucket{}
var _ RWRateQuota = &RWChildBucket{}
var _ periodBucket = &RWChildBucket{}

type RWChildBucket struct {
	s RWChildSubBucket
}

func (p *RWChildBucket) GetManager() *PeriodManager {
	return p.s.refiller.manager
}

func (p *RWChildBucket) CopyRWConfig() (read, write BucketConfig) {
	return p.s.stateR.config(), p.s.stateW.config()
}

func (p *RWChildBucket) TakeQuota(max int64) int64 {
	return p.s.refiller.TakeQuota(max, &p.s.stateR, p.s.refillR)
}

func (p *RWChildBucket) TakeQuotaNoWait(max int64) int64 {
	return p.s.refiller.TakeQuotaNoWait(max, &p.s.stateR, p.s.refillR)
}

func (p *RWChildBucket) TakeWriteQuota(max int64) int64 {
	return p.s.refiller.TakeQuota(max, &p.s.stateW, p.s.refillW)
}

func (p *RWChildBucket) TakeWriteQuotaNoWait(max int64) int64 {
	return p.s.refiller.TakeQuotaNoWait(max, &p.s.stateW, p.s.refillW)
}

func (p *RWChildBucket) TakeReadQuotaNoScale(max uint32) uint32 {
	return p.s.stateR.TakeQuotaNoScale(max, p.s.refillR)
}

func (p *RWChildBucket) TakeWriteQuotaNoScale(max uint32) uint32 {
	return p.s.stateW.TakeQuotaNoScale(max, p.s.refillW)
}

func (p *RWChildBucket) WriteBucket() RateQuota {
	return &p.s
}

type ParentRWBucket interface {
	TakeReadQuotaNoScale(max uint32) uint32
	TakeWriteQuotaNoScale(max uint32) uint32
	CopyRWConfig() (read, write BucketConfig)
}

type RWChildSubBucket struct {
	stateR   BucketState
	stateW   BucketState
	refiller PeriodRefiller

	quotaR RefillQuota
	quotaW RefillQuota
	parent ParentRWBucket
}

func (p *RWChildSubBucket) TakeQuota(max int64) int64 {
	return p.refiller.TakeQuota(max, &p.stateW, p.refillW)
}

func (p *RWChildSubBucket) TakeQuotaNoWait(max int64) int64 {
	return p.refiller.TakeQuotaNoWait(max, &p.stateW, p.refillW)
}

func (p *RWChildSubBucket) refillR(needed uint32) uint32 {
	x := p.refiller.GetRefillCount()
	p.stateW.ForceRefill(p.quotaW.Refill(needed, x, &p.stateW, p.parent.TakeWriteQuotaNoScale))
	return p.quotaR.Refill(needed, x, &p.stateR, p.parent.TakeReadQuotaNoScale)
}

func (p *RWChildSubBucket) refillW(needed uint32) uint32 {
	x := p.refiller.GetRefillCount()
	p.stateR.ForceRefill(p.quotaR.Refill(needed, x, &p.stateR, p.parent.TakeReadQuotaNoScale))
	return p.quotaW.Refill(needed, x, &p.stateW, p.parent.TakeWriteQuotaNoScale)
}
