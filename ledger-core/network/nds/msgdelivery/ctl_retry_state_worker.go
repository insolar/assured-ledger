// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newRetryRqWorker(sender *stateSender, parallel, postponed int) *retryRqWorker {
	switch {
	case parallel <= 0:
		panic(throw.IllegalValue())
	case postponed <= 0:
		panic(throw.IllegalValue())
	}

	return &retryRqWorker{
		sender:    sender,
		sema:      synckit.NewSemaphore(parallel),
		postponed: make([]postponedRq, postponed),
		marks:     &sender.marks,
	}
}

type retryRqWorker struct {
	sender *stateSender
	sema   synckit.Semaphore
	marks  *nodeRqMarks

	// circular buffer
	postponed   []postponedRq
	read, write int32

	wakeup chan struct{}
}

type postponedRq struct {
	rq       rqShipment
	repeatFn func(retries.RetryID)
}

func (p *retryRqWorker) runRetry() {
	p.wakeup = make(chan struct{}, 1)
	for {
		ok := false

		var oob rqShipment
		var state stateJob
		select {
		// NB! separate select ensures prioritized handling of channels
		case oob, ok = <-p.sender.oob:
		case state, ok = <-p.sender.states:
		default:
			select {
			case oob, ok = <-p.sender.oob:
			case state, ok = <-p.sender.states:

			case job, ok := <-p.sender.jobs:
				if !ok {
					return
				}
				p.processJob(job)
				continue
			case <-p.wakeup:
				if !p.isEmptyPostponed() {
					p.processPostponed()
				}
				continue
			}
		}

		switch {
		case !ok:
			return
		case !oob.isEmpty():
			p.processOoB(oob)
		case state.peer != nil:
			p.processState(state)
		}
	}
}

func (p *retryRqWorker) processOoB(rq rqShipment) {
	p.processRq(rq, p.sender.stages.AddHeadForRetry, true)
}

func (p *retryRqWorker) processRq(rq rqShipment, repeatFn func(retries.RetryID), sendNow bool) {
	captured, postponer := p.marks.mark(rq.id, p)
	switch {
	case !captured:
	case !p.sema.TryLock():
		p.marks.unmark(rq.id)
	default:
		go p._sendRq(rq, repeatFn, sendNow)
		return
	}

	if rq.isValid() {
		postponer.pushPostponed(rq, repeatFn)
	}
}

func (p *retryRqWorker) pushPostponed(rq rqShipment, repeatFn func(retries.RetryID)) {
	if prev := p.postponed[p.write]; !prev.rq.isEmpty() {
		if prev.repeatFn != nil {
			prev.repeatFn(retries.RetryID(prev.rq.id))
		} else {
			p.sender.RemoveByID(prev.rq.id)
		}
	}
	p.postponed[p.write] = postponedRq{rq, repeatFn}
	if p.write++; p.write >= int32(len(p.postponed)) {
		p.write = 0
	}
}

func (p *retryRqWorker) isEmptyPostponed() bool {
	return p.read == p.write
}

func (p *retryRqWorker) processPostponed() {
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

func (p *retryRqWorker) _processPostponedItem() {
	prq := p.postponed[p.read]
	p.postponed[p.read] = postponedRq{}
	p.read++
	switch {
	case prq.rq.peer != nil:
		p.processRq(prq.rq, prq.repeatFn, false)
	case prq.repeatFn != nil:
		p._processState(prq.rq.id.NodeID(), prq.repeatFn)
	}
}

func (p *retryRqWorker) _afterSend(nid uint32) {
	p.sema.Unlock()
	p.marks.unmarkNode(nid)

	select {
	case p.wakeup <- struct{}{}:
	default:
	}
}

func (p *retryRqWorker) _sendRq(rq rqShipment, repeatFn func(id retries.RetryID), sendNow bool) {
	defer p._afterSend(rq.id.NodeID())

	if sendNow {
		rq.peer.sendBodyRq(rq.id.ShortID())
	} else {
		rq.peer.addBodyRq(rq.id.ShortID())
	}
	repeatFn(retries.RetryID(rq.id))
}

func (p *retryRqWorker) processJob(job retryJob) {
	for _, id := range job.ids {
		switch rq, m := p.sender.get(ShipmentID(id)); m {
		case retries.KeepRetrying:
			p.processRq(rq, job.repeatFn, false)
		case retries.StopRetrying:
			job.repeatFn(id)
		}
	}
}

func (p *retryRqWorker) processState(state stateJob) {
	if !state.peer.isValid() {
		return
	}

	nid := uint32(state.peer.peerID)

	// NB! Here we wrap send as retry call to reuse the postpone buffer
	p._processState(nid, func(retries.RetryID) {
		defer p._afterSend(nid)
		state.peer.sendState(state.packet)
	})
}

func (p *retryRqWorker) _processState(nid uint32, fn func(retries.RetryID)) {
	captured, postponer := p.marks.markNode(nid, p)
	switch {
	case !captured:
	case !p.sema.TryLock():
		p.marks.unmarkNode(nid)
	default:
		go fn(0)
		return
	}

	postponer.pushPostponed(rqShipment{
		id: AsShipmentID(nid, 1),
	}, fn)
}
