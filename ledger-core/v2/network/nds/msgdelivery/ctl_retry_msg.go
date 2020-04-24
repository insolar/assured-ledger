// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type msgSender struct {
	stages retries.StagedController
	tracks ttlMap

	jobs chan retryJob
	oob  chan *msgShipment
}

func (p *msgSender) setConfig(batchSize uint, cycle time.Duration, c SenderConfig) {
	switch {
	case c.MaxPostponedPerWorker <= 0:
		panic(throw.IllegalValue())
	case c.ParallelPeersPerWorker <= 0:
		panic(throw.IllegalValue())
	case c.ParallelWorkers <= 0:
		panic(throw.IllegalValue())
	}

	p.init(c.FastQueue, c.RetryQueue)
	p.stages.InitStages(batchSize, calcPeriods(cycle, c.RetryIntervals))
}

func calcPeriods(timeCycle time.Duration, stages [retries.RetryStages]time.Duration) (periods [retries.RetryStages]int) {
	last := 1
	for i, ts := range stages {
		n := int(ts / timeCycle)
		if n < last {
			panic(throw.IllegalValue())
		}
		last = n
		periods[i] = n
	}
	return
}

func (p *msgSender) init(oobQueue, jobQueue int) {
	switch {
	case oobQueue <= 0:
		panic(throw.IllegalValue())
	case jobQueue <= 0:
		panic(throw.IllegalValue())
	case p.jobs != nil:
		panic(throw.IllegalState())
	}

	p.oob = make(chan *msgShipment, oobQueue)
	p.jobs = make(chan retryJob, jobQueue)
}

// ATTN! This method MUST return asap
func (p *msgSender) sendHead(msg *msgShipment, sz uint, currentCycle uint32) {
	p.tracks.put(msg, currentCycle)

	if !msg.isImmediateSend() || !p.sendHeadNoRetry(msg) {
		p.stages.Add(retries.RetryID(msg.id), sz, p)
	}
}

// ATTN! This method MUST return asap
func (p *msgSender) sendHeadNoRetry(msg *msgShipment) bool {
	select {
	case p.oob <- msg:
		return true
	default:
		return false
	}
}

// ATTN! This method MUST return asap
func (p *msgSender) sendBodyOnly(msg *msgShipment) {
	// TODO get rid of "go", use limiter somehow?
	go msg.sendBody()
}

func (p *msgSender) Retry(ids []retries.RetryID, repeatFn func(retries.RetryID)) {
	p.jobs <- retryJob{ids, repeatFn}
}

func (p *msgSender) CheckState(id retries.RetryID) retries.RetryState {
	if msg := p.get(ShipmentID(id)); msg != nil {
		return msg.getHeadRetryState()
	}
	// TODO move handling of retries.RemoveCompletely into Retry
	return retries.RemoveCompletely
}

func (p *msgSender) get(shid ShipmentID) *msgShipment {
	if msg := p.tracks.get(shid); msg != nil {
		return msg
	}
	return nil
}

func (p *msgSender) Remove(ids []retries.RetryID) {
	p.tracks.deleteAll(ids)
}

func (p *msgSender) nextTimeCycle() {
	p.stages.NextCycle(p)
}

func (p *msgSender) startWorker(parallel, postponed int) {
	if p.jobs == nil {
		panic(throw.IllegalState())
	}
	worker := newRetryMsgWorker(p, parallel, postponed)
	go worker.runRetry()
}

func (p *msgSender) stop() {
	close(p.oob)
}

/**********************************/

type nodeMarks struct {
	mutex sync.Mutex
	marks map[uint32]struct{}
}

func (p *nodeMarks) unmark(id ShipmentID) {
	p.unmarkNode(id.NodeID())
}

func (p *nodeMarks) mark(id ShipmentID) bool {
	return p.markNode(id.NodeID())
}

func (p *nodeMarks) markNode(nid uint32) bool {
	p.mutex.Lock()
	if _, ok := p.marks[nid]; ok {
		p.mutex.Unlock()
		return false
	}
	p.marks[nid] = struct{}{}
	p.mutex.Unlock()
	return true
}

func (p *nodeMarks) unmarkNode(nid uint32) {
	p.mutex.Lock()
	delete(p.marks, nid)
	p.mutex.Unlock()
}
