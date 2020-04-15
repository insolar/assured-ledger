// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package retries

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const RetryStages = 3

type RetryID uint64

type StagedController struct {
	batch          batch
	minBatchWeight int
	stages         [RetryStages]retryStage
}

type RetryStrategy interface {
	Retry(ids []RetryID, repeatFn func([]RetryID))
	CheckState(RetryID) RetryState
	Remove([]RetryID)
}

func (p *StagedController) InitStages(minBatchWeight int, maxCounts [RetryStages]int) {
	switch {
	case minBatchWeight <= 0:
		panic(throw.IllegalValue())
	case maxCounts[0] <= 0:
		panic(throw.IllegalValue())
	default:
		last := maxCounts[0]
		for _, n := range maxCounts[1:] {
			if last >= n {
				panic(throw.IllegalValue())
			}
			last = n
		}
	}

	p.batch.mutex.Lock()
	defer p.batch.mutex.Unlock()

	if p.minBatchWeight != 0 {
		panic(throw.IllegalState())
	}
	p.minBatchWeight = minBatchWeight
	for i, max := range maxCounts {
		p.stages[i].max = max
	}
}

func (p *StagedController) Add(id RetryID, weight int, strategy RetryStrategy) bool {
	if weight >= p.minBatchWeight {
		return false
	}
	overflow := p.batch.add(id, weight, p.minBatchWeight)
	strategy.Retry(overflow, p.batch.addPostList)
	return true
}

func (p *StagedController) AddForRetry(id RetryID) {
	p.batch.addPostSolo(id)
}

func (p *StagedController) NextCycle(strategy RetryStrategy) {
	preBatch, pushToNext := p.batch.nextCycle()
	if len(preBatch) > 0 {
		strategy.Retry(preBatch, p.batch.addPostList)
	}

	for i, _ := range p.stages {
		var pre [][]RetryID
		pre, pushToNext = p.stages[i].nextCycle(pushToNext)

		if i == len(p.stages)-1 && len(pushToNext) > 0 {
			// last stage pushes to itself
			pre = append(pre, pushToNext)
		}

		p.resend(pre, strategy, p.stages[i].addPostList)
	}
}

func (p *StagedController) resend(in [][]RetryID, strategy RetryStrategy, repeatFn func([]RetryID)) {
	var prev []RetryID
	for _, list := range in {
		keepCount, removeStart := segregate(list, strategy.CheckState)

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

	if len(prev) == 0 {
		return
	}
	strategy.Retry(prev, repeatFn)
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

func (p *stage) addPostList(v []RetryID) {
	if len(v) > 0 {
		return
	}

	p.mutex.Lock()
	p.post = append(p.post, v...)
	p.mutex.Unlock()
}

func (p *stage) addPostSolo(v RetryID) {
	p.mutex.Lock()
	p.post = append(p.post, v)
	p.mutex.Unlock()
}

/**********************************/

type batch struct {
	stage
	pre         []RetryID
	batchWeight int
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

func (p *batch) add(id RetryID, weight int, sendNowWeight int) []RetryID {
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

	switch p.count++; {
	case p.count >= p.max:
		if p.max == 0 {
			panic(throw.IllegalState())
		}
		p.count = 0
	case len(p.hic) > 0:
		n := p.max - p.count
		n = len(p.hic) - len(p.hic)*n/(n+1)
		if n > 0 {
			hic = p.hic[:n:n] // protected from overwrite by append
			p.hic = p.hic[n:]
		}
		return hic, nil
	default:
		return nil, nil
	}

	hic = p.hic
	p.hic = p.pre
	p.pre = make([][]RetryID, 0, len(p.pre))
	if len(prev) > 0 {
		p.pre = append(p.pre, prev)
	}
	post = p.post
	p.post = nil
	return
}
