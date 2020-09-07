// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
		marks:     nodeMarks{marks: make(map[uint32]struct{}, parallel)},
		postponed: make([]postponedMsg, postponed),
	}
}

type retryMsgWorker struct {
	sender *msgSender
	sema   synckit.Semaphore
	marks  nodeMarks

	// circular buffer
	postponed   []postponedMsg
	read, write int32

	wakeup chan struct{}
}

type retryJob struct {
	ids      []retries.RetryID
	repeatFn func(retries.RetryID)
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
		case oob != nil:
			p.processOoB(oob)
		default:
			for _, id := range job.ids {
				p.processMsg(p.sender.get(ShipmentID(id)), job.repeatFn)
			}
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

func (p *retryMsgWorker) processOoB(msg *msgShipment) {
	switch {
	case p.marks.mark(msg.id):
		p.sema.Lock()
		go p.sendHead(msg, p.sender.stages.AddHeadForRetry)
	case msg.canSendHead():
		p.pushPostponed(msg, p.sender.stages.AddHeadForRetry)
	}
}

func (p *retryMsgWorker) processMsg(msg *msgShipment, repeatFn func(retries.RetryID)) {
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

func (p *retryMsgWorker) pushPostponed(msg *msgShipment, repeatFn func(retries.RetryID)) {
	if msg == nil {
		return
	}
	if prev := p.postponed[p.write]; prev.msg != nil {
		if prev.msg.canSendHead() && prev.repeatFn != nil {
			prev.repeatFn(retries.RetryID(msg.id))
		} else {
			prev.msg.markCancel()
		}
	}
	p.postponed[p.write] = postponedMsg{msg, repeatFn}
	if p.write++; p.write >= int32(len(p.postponed)) {
		p.write = 0
	}
}

func (p *retryMsgWorker) isEmptyPostponed() bool {
	return p.read == p.write
}

func (p *retryMsgWorker) processPostponed() {
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

func (p *retryMsgWorker) _processPostponedItem() {
	pmsg := p.postponed[p.read]
	p.postponed[p.read] = postponedMsg{}
	p.read++
	p.processMsg(pmsg.msg, pmsg.repeatFn)
}
