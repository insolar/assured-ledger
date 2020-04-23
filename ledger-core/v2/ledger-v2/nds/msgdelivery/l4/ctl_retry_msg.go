// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type msgSender struct {
	stages retries.StagedController
	tracks msgMap

	jobs chan retryJob
	oob  chan *msgShipment
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
func (p *msgSender) sendHead(msg *msgShipment, sz uint) {
	p.tracks.put(msg)

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
	if msg := p.getHeads(ShipmentID(id)); msg != nil {
		return msg.getHeadRetryState()
	}
	// TODO move handling of retries.RemoveCompletely into Retry
	return retries.RemoveCompletely
}

func (p *msgSender) getHeads(shid ShipmentID) *msgShipment {
	return p.tracks.get(shid)
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

func (p *msgSender) NextTimeCycle() {
	p.stages.NextCycle(p)
}

func (p *msgSender) startWorker(parallel int) {
	if p.jobs == nil {
		panic(throw.IllegalState())
	}
	worker := newRetryMsgWorker(p, parallel)
	go worker.runRetry()
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
