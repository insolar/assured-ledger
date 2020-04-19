// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

type retrySender struct {
	stages retries.StagedController
	tracks msgMap

	jobs chan retryJob
	oob  chan *msgShipment
}

// ATTN! This method MUST return asap
func (p *retrySender) sendHeadNoWait(msg *msgShipment, sz uint) {
	p.tracks.put(msg)

	if !msg.isImmediateSend() || !p.sendHeadNoRetryNoWait(msg) {
		p.stages.Add(retries.RetryID(msg.id), sz, p)
	}
}

// ATTN! This method MUST return asap
func (p *retrySender) sendHeadNoRetryNoWait(msg *msgShipment) bool {
	select {
	case p.oob <- msg:
		return true
	default:
		return false
	}
}

// ATTN! This method MUST return asap
func (p *retrySender) sendBodyNoWait(msg *msgShipment) {
	// TODO get rid of "go", use limiter somehow?
	go msg.sendBody()
}

func (p *retrySender) Retry(ids []retries.RetryID, repeatFn func(retries.RetryID)) {
	p.jobs <- retryJob{ids, repeatFn}
}

func (p *retrySender) CheckState(id retries.RetryID) retries.RetryState {
	if msg := p.getHeads(ShipmentID(id)); msg != nil {
		return msg.getHeadRetryState()
	}
	return retries.RemoveCompletely
}

func (p *retrySender) getHeads(shid ShipmentID) *msgShipment {
	return p.tracks.get(shid)
}

func (p *retrySender) get(shid ShipmentID) *msgShipment {
	if msg := p.tracks.get(shid); msg != nil {
		return msg
	}
	return nil
}

func (p *retrySender) Remove(ids []retries.RetryID) {
	p.tracks.deleteAll(ids)
}

func (p *retrySender) NextTimeCycle() {
	p.stages.NextCycle(p)
}

func (p *retrySender) startWorker(parallel int) {
	worker := retryWorker{
		sender:    p,
		sema:      synckit.NewSemaphore(parallel),
		marks:     nodeMarks{marks: make(map[uint32]struct{}, parallel)},
		postponed: make([]postponedMsg, 0, 100),
	}
	go worker.runRetry()
}

/**********************************/

type retryJob struct {
	ids      []retries.RetryID
	repeatFn func(retries.RetryID)
}

type postponedMsg struct {
	msg      *msgShipment
	repeatFn func(retries.RetryID)
}

/**********************************/

type nodeMarks struct {
	mutex sync.Mutex
	marks map[uint32]struct{}
}

func (p *nodeMarks) unmark(id ShipmentID) {
	p.mutex.Lock()
	delete(p.marks, id.NodeID())
	p.mutex.Unlock()
}

func (p *nodeMarks) mark(id ShipmentID) bool {
	p.mutex.Lock()
	nid := id.NodeID()
	if _, ok := p.marks[nid]; ok {
		p.mutex.Unlock()
		return false
	}
	p.marks[nid] = struct{}{}
	p.mutex.Unlock()
	return true
}

/**********************************/

type retryWorker struct {
	sender *retrySender
	sema   synckit.Semaphore
	marks  nodeMarks

	// circular buffer
	postponed   []postponedMsg
	read, write int32

	wakeup chan struct{}
}

func (p *retryWorker) runRetry() {
	p.wakeup = make(chan struct{}, 1)
	for {
		job := retryJob{}
		ok := false

		var oob *msgShipment
		select {
		case oob, ok = <-p.sender.oob:
		default:
			select {
			case oob, ok = <-p.sender.oob:
			case job, ok = <-p.sender.jobs:
			case <-p.wakeup:
				if p.isEmptyPostponed() {
					continue
				}
			}
		}

		switch {
		case !ok:
			return
		case oob != nil:
			p.processOoB(oob)
		case !p.isEmptyPostponed():
			p.processPostponed()
		}

		for _, id := range job.ids {
			p.processMsg(p.sender.get(ShipmentID(id)), job.repeatFn)
		}
	}
}

func (p *retryWorker) sendHead(msg *msgShipment, repeatFn func(retries.RetryID)) {
	defer func() {
		p.sema.Unlock()
		p.marks.unmark(msg.id)

		select {
		case p.wakeup <- struct{}{}:
		default:
		}
	}()

	if msg.sendHead() {
		repeatFn(retries.RetryID(msg.id))
	}
}

func (p *retryWorker) processOoB(msg *msgShipment) {
	switch {
	case p.marks.mark(msg.id):
		p.sema.Lock()
		go p.sendHead(msg, p.sender.stages.AddHeadForRetry)
	case msg.canSendHead():
		p.pushPostponed(msg, p.sender.stages.AddHeadForRetry)
	}
}

func (p *retryWorker) processMsg(msg *msgShipment, repeatFn func(retries.RetryID)) {
	switch {
	case msg == nil:
		return
	case !p.marks.mark(msg.id):
		//
	case !p.sema.TryLock():
		p.marks.unmark(msg.id)
	default:
		go p.sendHead(msg, repeatFn)
		return
	}

	if msg.canSendHead() {
		p.pushPostponed(msg, repeatFn)
	}
}

func (p *retryWorker) pushPostponed(msg *msgShipment, repeatFn func(retries.RetryID)) {
	if msg == nil {
		return
	}
	if prev := p.postponed[p.write]; prev.msg != nil {
		prev.msg.markCancel()
	}
	p.postponed[p.write] = postponedMsg{msg, repeatFn}
	if p.write++; p.write >= int32(len(p.postponed)) {
		p.write = 0
	}
}

func (p *retryWorker) isEmptyPostponed() bool {
	return p.read == p.write
}

func (p *retryWorker) processPostponed() {
	lastWrite := p.write
	if p.read > p.write {
		for p.read < int32(len(p.postponed)) {
			p._processPostponedItem()
		}
		p.read = 0
	}
	for p.read < lastWrite {
		p._processPostponedItem()
	}
}

func (p *retryWorker) _processPostponedItem() {
	pmsg := p.postponed[p.read]
	p.postponed[p.read] = postponedMsg{}
	p.read++
	p.processMsg(pmsg.msg, pmsg.repeatFn)
}
