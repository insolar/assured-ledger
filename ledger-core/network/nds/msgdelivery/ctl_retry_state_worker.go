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
		postponed: make(chan postponedRq, postponed),
		marks:     &sender.marks,
	}
}

type retryRqWorker struct {
	sender *stateSender
	sema   synckit.Semaphore
	marks  *nodeRqMarks

	// buffer
	postponed      chan postponedRq
	postponedRead  <- chan postponedRq
	postponedLimit uint32

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
			case postponedItem := <- p.postponedRead:
				p.processPostponed(postponedItem)
				continue
			case <-p.wakeup:
				p.enablePostponedProcessing()
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

func (p *retryRqWorker) processRq(rq rqShipment, repeatFn func(retries.RetryID), sendNow bool) bool {
	captured, postponer := p.marks.mark(rq.id, p)
	switch {
	case !captured:
	case !p.sema.TryLock():
		p.marks.unmark(rq.id)
	default:
		go p._sendRq(rq, repeatFn, sendNow)
		return true
	}

	if rq.isValid() {
		postponer.pushPostponed(rq, repeatFn)
		return false
	}

	return true
}

func (p *retryRqWorker) pushPostponed(rq rqShipment, repeatFn func(retries.RetryID)) {
	select {
	case p.postponed <- postponedRq{rq, repeatFn}:
		return
	default:
		if repeatFn != nil {
			repeatFn(retries.RetryID(rq.id))
		} else {
			p.sender.RemoveByID(rq.id)
		}
	}
}

func (p *retryRqWorker) processPostponed(rq postponedRq) {
	switch {
	case p._processPostponed(rq):
	case p.postponedLimit > 1:
		p.postponedLimit--
	default:
		p.postponedLimit = 0
		p.disablePostponedProcessing()
	}
}

func (p *retryRqWorker) _processPostponed(prq postponedRq) bool {
	switch {
	case prq.rq.peer != nil:
		return p.processRq(prq.rq, prq.repeatFn, false)
	case prq.repeatFn != nil:
		return p._processState(prq.rq.id.NodeID(), prq.repeatFn)
	}
	return true
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

func (p *retryRqWorker) _processState(nid uint32, fn func(retries.RetryID)) bool {
	captured, postponer := p.marks.markNode(nid, p)
	switch {
	case !captured:
	case !p.sema.TryLock():
		p.marks.unmarkNode(nid)
	default:
		go fn(0)
		return true
	}

	postponer.pushPostponed(rqShipment{
		id: AsShipmentID(nid, 1),
	}, fn)
	return false
}

func (p *retryRqWorker) enablePostponedProcessing() {
	if p.postponedRead == nil {
		p.postponedLimit = uint32(len(p.postponed)) + 1
		p.postponedRead = p.postponed
	}
}

func (p *retryRqWorker) disablePostponedProcessing() {
	p.postponedRead = nil
}
