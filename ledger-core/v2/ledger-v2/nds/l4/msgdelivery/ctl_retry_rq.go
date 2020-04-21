// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type rqSender struct {
	stages retries.StagedController

	oob  chan rqShipment
	jobs chan retryJob

	mutex    sync.Mutex
	requests map[ShipmentID]rqShipment
	suspends map[ShipmentID]struct{}
}

func (p *rqSender) Add(rq rqShipment) error {
	if err := p.put(rq); err != nil {
		return err
	}

	p.oob <- rq
	return nil
}

func (p *rqSender) put(rq rqShipment) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ok := p.requests[rq.id]; ok {
		return throw.FailHere("duplicate request")
	}
	p.requests[rq.id] = rq
	return nil
}

func (p *rqSender) get(id ShipmentID) (rqShipment, retries.RetryState) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	rq, ok := p.requests[id]
	if !ok {
		return rq, retries.RemoveCompletely
	}
	if _, ok = p.suspends[id]; ok {
		return rq, retries.StopRetrying
	}
	return rq, retries.KeepRetrying
}

func (p *rqSender) RemoveRq(peer *DeliveryPeer, id ShortShipmentID) (rqShipment, bool) {
	return p.RemoveByID(AsShipmentID(uint32(peer.peerID), id))
}

func (p *rqSender) RemoveByID(sid ShipmentID) (rqShipment, bool) {
	p.mutex.Lock()
	rq, ok := p.requests[sid]
	if ok {
		delete(p.suspends, sid)
		delete(p.requests, sid)
	}
	p.mutex.Unlock()

	if !ok || rq.request.Cancel.IsCancelled() {
		return rqShipment{}, false
	}
	return rq, ok
}

func (p *rqSender) suspendRetry(id ShipmentID) func() {
	p.mutex.Lock()
	if _, ok := p.requests[id]; !ok {
		p.mutex.Unlock()
		return nil
	}

	p.suspends[id] = struct{}{}
	p.mutex.Unlock()

	return func() {
		p.mutex.Lock()
		delete(p.suspends, id)
		p.mutex.Unlock()
	}
}

func (p *rqSender) NextTimeCycle() {
	p.stages.NextCycle(p)
}

func (p *rqSender) Retry(ids []retries.RetryID, repeatFn func(retries.RetryID)) {

	keepCount, removeStart := p.retry(ids)

	for _, id := range ids[keepCount:removeStart] {
		repeatFn(id)
	}

	if keepCount > 0 {
		p.jobs <- retryJob{ids: ids[:keepCount], repeatFn: repeatFn}
	}
}

func (p *rqSender) retry(ids []retries.RetryID) (keepCount, removeStart int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return retries.Segregate(ids, func(id retries.RetryID) retries.RetryState {
		sid := ShipmentID(id)
		switch rq, ok := p.requests[sid]; {
		case !ok:
			return retries.RemoveCompletely
		case rq.isExpired() || !rq.peer.isValid():
			delete(p.requests, sid)
			delete(p.suspends, sid)
			return retries.RemoveCompletely
		}
		if _, ok := p.suspends[sid]; ok {
			return retries.StopRetrying
		}
		return retries.KeepRetrying
	})
}

func (p *rqSender) CheckState(retries.RetryID) (r retries.RetryState) {
	return retries.KeepRetrying
}

func (p *rqSender) Remove(ids []retries.RetryID) {
	p.mutex.Lock()
	for _, id := range ids {
		sid := ShipmentID(id)
		delete(p.requests, sid)
		delete(p.suspends, sid)
	}
	p.mutex.Unlock()
}

func (p *rqSender) startWorker(parallel int) {
	worker := newRetryRqWorker(p, parallel)
	go worker.runRetry()
}
