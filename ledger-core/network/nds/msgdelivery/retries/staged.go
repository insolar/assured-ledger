// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package retries

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const RetryStages = 3

type RetryID uint64

type StagedController struct {
	batchHeads batch
	stages     [RetryStages]retryStage

	minHeadBatchWeight uint
}

type RetryStrategy interface {
	Retry(ids []RetryID, repeatFn func(RetryID))
	CheckState(RetryID) RetryState
	Remove([]RetryID)
}

// for initialization only
func (p *StagedController) InitStages(minHeadBatchWeight uint, periods [RetryStages]int) {
	switch {
	case minHeadBatchWeight == 0:
		panic(throw.IllegalValue())
	case periods[0] <= 0:
		panic(throw.IllegalValue())
	default:
		last := periods[0]
		for _, n := range periods[1:] {
			if last >= n {
				panic(throw.IllegalValue())
			}
			last = n
		}
	}

	// p.batchHeads.mutex.Lock()
	// defer p.batchHeads.mutex.Unlock()

	p.minHeadBatchWeight = minHeadBatchWeight
	for i, max := range periods {
		p.stages[i].max = max
	}
}

func (p *StagedController) Add(id RetryID, weight uint, strategy RetryStrategy) {
	overflow := p.batchHeads.add(id, weight, p.minHeadBatchWeight)
	if len(overflow) > 0 {
		strategy.Retry(overflow, p.batchHeads.addSent)
	}
}

func (p *StagedController) AddHeadForRetry(id RetryID) {
	p.batchHeads.addSent(id)
}

func (p *StagedController) NextCycle(strategy RetryStrategy) {
	preBatch, pushToNext := p.batchHeads.nextCycle()
	if len(preBatch) > 0 {
		strategy.Retry(preBatch, p.batchHeads.addSent)
	}

	for i, maxStage := 0, len(p.stages)-1; i <= maxStage; i++ {
		var resend [][]RetryID
		resend, pushToNext = p.stages[i].nextCycle(pushToNext)

		if i < maxStage {
			p.resend(resend, strategy, p.stages[i].addSent)
			continue
		}

		p.resend(resend, strategy, p.stages[i-1].addSent)
		if i == maxStage && len(pushToNext) > 0 {
			// last stage pushes to its own input
			p.stages[i].addPreList(pushToNext)
		}
	}
}

func (p *StagedController) resend(in [][]RetryID, strategy RetryStrategy, repeatFn func(RetryID)) {
	var prev []RetryID
	for _, list := range in {
		keepCount, removeStart := Segregate(list, strategy.CheckState)

		if rm := list[removeStart:]; len(rm) > 0 {
			strategy.Remove(list[removeStart:])
		}

		switch free := cap(prev) - len(prev); {
		case keepCount == 0:
			//
		case keepCount >= len(prev) || keepCount > free:
			if len(prev) > 0 {
				strategy.Retry(prev, repeatFn)
			}
			prev = list[:keepCount]

		case keepCount == free:
			prev = append(prev, list[:keepCount]...)
			strategy.Retry(prev, repeatFn)
			prev = nil

		default:
			prev = append(prev, list[:keepCount]...)
		}
	}

	if len(prev) > 0 {
		strategy.Retry(prev, repeatFn)
	}
}

type RetryState uint8

const (
	KeepRetrying RetryState = iota
	StopRetrying
	RemoveCompletely
)

/**********************************/

type stage struct {
	mutex sync.Mutex
	post  []RetryID
}

func (p *stage) addSent(v RetryID) {
	p.mutex.Lock()
	p.post = append(p.post, v)
	p.mutex.Unlock()
}

/**********************************/

type batch struct {
	stage
	pre         []RetryID
	batchWeight uint
}

func (p *batch) nextCycle() (pre []RetryID, post []RetryID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.pre) > 0 {
		pre = p.pre
		p.pre = make([]RetryID, 0, len(pre))
	} else {
		p.pre = nil
	}
	post = p.post
	p.post = nil
	return
}

func (p *batch) add(id RetryID, weight uint, sendNowWeight uint) []RetryID {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.batchWeight += weight
	p.pre = append(p.pre, id)

	if p.batchWeight < sendNowWeight {
		return nil
	}

	pre := p.pre
	p.batchWeight = 0
	p.pre = make([]RetryID, 0, len(pre))
	return pre
}

/**********************************/

type retryStage struct {
	stage
	pre   [][]RetryID
	hic   [][]RetryID
	count int
	max   int
}

func (p *retryStage) nextCycle(prev []RetryID) (hic [][]RetryID, post []RetryID) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.count++
	if p.count < p.max {
		if len(prev) > 0 {
			p.pre = append(p.pre, prev)
		}
		if len(p.hic) == 0 {
			return nil, nil
		}
		n := p.max - p.count
		n = len(p.hic) - len(p.hic)*n/(n+1)
		if n > 0 {
			hic = p.hic[:n:n] // protected from overwrite by append
			p.hic = p.hic[n:]
		}
		return hic, nil
	}

	if p.max == 0 {
		panic(throw.IllegalState())
	}
	p.count = 0

	hic = p.hic
	p.hic = p.pre
	if n := len(p.pre); n > 0 {
		p.pre = make([][]RetryID, 0, n)
	}
	if len(prev) > 0 {
		p.pre = append(p.pre, prev)
	}
	post = p.post
	p.post = nil
	return hic, post
}

func (p *retryStage) addPreList(v []RetryID) {
	if len(v) > 0 {
		return
	}

	p.mutex.Lock()
	p.pre = append(p.pre, v)
	p.mutex.Unlock()
}
