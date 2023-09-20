package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
)

func newRetryMsgWorker(sender *msgSender, parallel, postponed int) *retryMsgWorker {
	switch {
	case parallel <= 0:
		panic(throw.IllegalValue())
	case postponed <= 0:
		panic(throw.IllegalValue())
	}

	return &retryMsgWorker{
		sender:    sender,
		sema:      synckit.NewSemaphore(parallel),
		postponed: make(chan postponedMsg, postponed),
		marks:     &sender.marks,
	}
}

type retryMsgWorker struct {
	sender *msgSender
	sema   synckit.Semaphore
	marks  *nodeMsgMarks

	// buffer
	postponed     chan postponedMsg
	postponedRead <- chan postponedMsg
	postponedLimit uint32

	wakeup chan struct{}
}

const minRetryBulkSize = 32

type retryJob struct {
	ids      []retries.RetryID
	repeatFn func(retries.RetryID)
	bulkFn   func([]retries.RetryID)
}

type postponedMsg struct {
	msg      *msgShipment
	repeatFn func(retries.RetryID)
}

func (p *retryMsgWorker) runRetry() {
	p.wakeup = make(chan struct{}, 1)
	for {
		job := retryJob{}
		ok := false

		var oob *msgShipment
		select {
		// NB! separate select ensures prioritized handling of channels
		case oob, ok = <-p.sender.oob:
		default:
			select {
			case oob, ok = <-p.sender.oob:
			case job, ok = <-p.sender.jobs:
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
		case oob != nil:
			p.processOoB(oob)
		default:
			p.processRetryJob(job)
		}
	}
}

func (p *retryMsgWorker) processRetryJob(job retryJob) {
	for i, id := range job.ids {
		switch {
		case p.processMsg(p.sender.get(ShipmentID(id)), job.repeatFn):
		case !p.sema.IsFull():
		case i + minRetryBulkSize <= len(job.ids):
			job.bulkFn(job.ids[i+1:])
			return
		}
	}
}

func (p *retryMsgWorker) _afterSend(id ShipmentID) {
	p.sema.Unlock()
	p.marks.unmark(id)

	select {
	case p.wakeup <- struct{}{}:
	default:
	}
}

func (p *retryMsgWorker) sendHead(msg *msgShipment, repeatFn func(retries.RetryID)) {
	defer p._afterSend(msg.id)

	if msg.sendHead() {
		repeatFn(retries.RetryID(msg.id))
	}
}

func noRetryFn(retries.RetryID) {}

func (p *retryMsgWorker) processOoB(msg *msgShipment) {
	if msg.isFireAndForget() {
		p.processMsg(msg, noRetryFn)
	} else {
		p.processMsg(msg, p.sender.stages.AddHeadForRetry)
	}
}

func (p *retryMsgWorker) processMsg(msg *msgShipment, repeatFn func(retries.RetryID)) bool {
	if msg == nil {
		return true
	}

	captured, postponer := p.marks.mark(msg.id, p)
	switch {
	case !captured:
	case !p.sema.TryLock():
		p.marks.unmark(msg.id)
	default:
		go p.sendHead(msg, repeatFn)
		return true
	}

	if msg.canSendHead() {
		postponer.pushPostponed(msg, repeatFn)
		return false
	}
	return true
}

func (p *retryMsgWorker) pushPostponed(msg *msgShipment, repeatFn func(retries.RetryID)) {
	if msg == nil {
		return
	}

	select {
	case p.postponed <- postponedMsg{msg, repeatFn}:
		return
	default:
		if msg.canSendHead() && repeatFn != nil {
			repeatFn(retries.RetryID(msg.id))
		} else {
			msg.markCancel()
		}
	}
}

func (p *retryMsgWorker) processPostponed(pmsg postponedMsg) {
	switch {
	case p.processMsg(pmsg.msg, pmsg.repeatFn):
	case p.postponedLimit > 1:
		p.postponedLimit--
	default:
		p.postponedLimit = 0
		p.disablePostponedProcessing()
	}
}

func (p *retryMsgWorker) enablePostponedProcessing() {
	if p.postponedRead == nil {
		p.postponedLimit = uint32(len(p.postponed)) + 1
		p.postponedRead = p.postponed
	}
}

func (p *retryMsgWorker) disablePostponedProcessing() {
	p.postponedRead = nil
}

