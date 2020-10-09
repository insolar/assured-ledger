// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
)

type msgSender struct {
	stages retries.StagedController
	tracks ttlMap

	marks  nodeMsgMarks

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

	if !msg.isImmediateSend() || !p.sendHeadNow(msg) {
		p.stages.Add(retries.RetryID(msg.id), sz, p)
	}
}

// ATTN! This method MUST return asap
func (p *msgSender) sendHeadNow(msg *msgShipment) bool {
	select {
	case p.oob <- msg:
		return true
	default:
		return false
	}
}

// ATTN! This method should return asap
func (p *msgSender) sendBodyOnly(msg *msgShipment) {
	// TODO use "go" with limiter somehow?
	msg.sendBody()
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
	return p.tracks.get(shid)
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

type nodeMsgMarks struct {
	mutex sync.Mutex
	marks map[uint32]*retryMsgWorker
}

func (p *nodeMsgMarks) unmark(id ShipmentID) {
	p.unmarkNode(id.NodeID())
}

func (p *nodeMsgMarks) mark(id ShipmentID, w *retryMsgWorker) (bool, *retryMsgWorker) {
	return p.markNode(id.NodeID(), w)
}

func (p *nodeMsgMarks) markNode(nid uint32, w *retryMsgWorker) (bool, *retryMsgWorker) {
	if w == nil {
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.marks == nil {
		p.marks = map[uint32]*retryMsgWorker{}
	} else if owner, ok := p.marks[nid]; ok {
		return false, owner
	}

	p.marks[nid] = w
	return true, w
}

func (p *nodeMsgMarks) unmarkNode(nid uint32) {
	p.mutex.Lock()
	delete(p.marks, nid)
	p.mutex.Unlock()
}
