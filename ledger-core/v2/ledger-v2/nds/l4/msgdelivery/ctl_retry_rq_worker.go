// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

func newRetryRqWorker(sender *rqSender, parallel int) *retryRqWorker {
	return &retryRqWorker{
		sender:    sender,
		sema:      synckit.NewSemaphore(parallel),
		marks:     nodeMarks{marks: make(map[uint32]struct{}, parallel)},
		postponed: make([]postponedRq, 0, 100),
	}
}

type retryRqWorker struct {
	sender *rqSender
	sema   synckit.Semaphore
	marks  nodeMarks

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
		job := retryJob{}
		ok := false

		var oob rqShipment
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
		case !oob.isEmpty():
			p.processOoB(oob)
		case !p.isEmptyPostponed():
			p.processPostponed()
		}

		for _, id := range job.ids {
			switch rq, m := p.sender.get(ShipmentID(id)); m {
			case retries.KeepRetrying:
				p.processRq(rq, job.repeatFn)
			case retries.StopRetrying:
				job.repeatFn(id)
			}
		}
	}
}

func (p *retryRqWorker) processOoB(rq rqShipment) {
	switch {
	case p.marks.mark(rq.id):
		p.sema.Lock()
		go p.sendRq(rq, p.sender.stages.AddHeadForRetry, true)
	case rq.isValid():
		p.pushPostponed(rq, p.sender.stages.AddHeadForRetry)
	}
}

func (p *retryRqWorker) processRq(rq rqShipment, repeatFn func(retries.RetryID)) {
	switch {
	case !p.marks.mark(rq.id):
		//
	case !p.sema.TryLock():
		p.marks.unmark(rq.id)
	default:
		go p.sendRq(rq, repeatFn, false)
		return
	}

	if rq.isValid() {
		p.pushPostponed(rq, repeatFn)
	}
}

func (p *retryRqWorker) pushPostponed(rq rqShipment, repeatFn func(retries.RetryID)) {
	if prev := p.postponed[p.write]; !prev.rq.isEmpty() {
		p.sender.RemoveByID(prev.rq.id)
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
	p.processRq(prq.rq, prq.repeatFn)
}

func (p *retryRqWorker) sendRq(rq rqShipment, repeatFn func(id retries.RetryID), sendNow bool) {
	defer func() {
		p.sema.Unlock()
		p.marks.unmark(rq.id)

		select {
		case p.wakeup <- struct{}{}:
		default:
		}
	}()

	if !sendNow {
		rq.peer.addBodyRq(rq.id.ShortID())
	} else {
		rq.peer.sendBodyRq(rq.id.ShortID())
	}
	repeatFn(retries.RetryID(rq.id))
}
