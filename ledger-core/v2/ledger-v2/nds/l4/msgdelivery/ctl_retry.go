// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
)

type retrySender struct {
	stages retries.StagedController
	heads  msgMap
	bodies msgMap
}

// ATTN! This method MUST return asap
func (p *retrySender) sendHeadNoWait(msg *msgShipment, sz uint) {
	p.heads.put(msg)

	rid := retries.RetryID(msg.id)
	if msg.isImmediateSend() {
		p.sendAsap(msg, p.stages.AddHeadForRetry)
		return
	}
	p.stages.Add(rid, sz, p)
}

// ATTN! This method MUST return asap
func (p *retrySender) sendBodiesNoWait(msgs ...*msgShipment) {
	p.heads.deleteMsgList(msgs)
	p.bodies.putAll(msgs)

	// TODO "go" is insecure because sender can potentially flood us
	go msgs[0].sendBody(false)
}

func (p *retrySender) Retry(ids []retries.RetryID, repeatFn func([]retries.RetryID)) {
	p.queueRetry(ids, repeatFn)
	//j := 0
	//for i, id := range ids {
	//	switch msg := p.getHeads(ShipmentID(id)); {
	//	case msg == nil:
	//		switch msg := p.getBodies(ShipmentID(id)); {
	//		case msg == nil:
	//			continue
	//		case msg.getBodyRetryState() == retries.KeepRetrying:
	//			msg.sendBody(true)
	//		default:
	//			continue
	//		}
	//	case msg.getHeadRetryState() == retries.KeepRetrying:
	//		msg.sendHead(true)
	//	default:
	//		continue
	//	}
	//	if j != i {
	//		ids[j] = id
	//	}
	//	j++
	//}
	//
	//if j > 0 {
	//	repeatFn(ids[:j])
	//}
}

func (p *retrySender) CheckState(id retries.RetryID) retries.RetryState {
	if msg := p.getHeads(ShipmentID(id)); msg != nil {
		return msg.getHeadRetryState()
	}
	if msg := p.getBodies(ShipmentID(id)); msg != nil {
		return msg.getBodyRetryState()
	}
	return retries.RemoveCompletely
}

func (p *retrySender) getHeads(shid ShipmentID) *msgShipment {
	return p.heads.get(shid)
}

func (p *retrySender) getBodies(shid ShipmentID) *msgShipment {
	return p.bodies.get(shid)
}

func (p *retrySender) get(shid ShipmentID) *msgShipment {
	if msg := p.heads.get(shid); msg != nil {
		return msg
	}
	return p.bodies.get(shid)
}

func (p *retrySender) Remove(ids []retries.RetryID) {
	ids = p.heads.deleteAllAndShrink(ids)
	p.bodies.deleteAll(ids)
}

func (p *retrySender) NextTimeCycle() {
	p.stages.NextCycle(p)
}

func (p *retrySender) sendAsap(msg *msgShipment, repeatFn func(id retries.RetryID)) {

}

func (p *retrySender) queueRetry(ids []retries.RetryID, repeatFn func([]retries.RetryID)) {

}

/**********************************/

type msgMap struct {
	mx sync.RWMutex
	mp map[ShipmentID]*msgShipment
}

func (p *msgMap) put(msg *msgShipment) {
	p.mx.Lock()
	p.mp[msg.id] = msg
	p.mx.Unlock()
}

func (p *msgMap) putAll(msgs []*msgShipment) {
	if len(msgs) == 0 {
		return
	}
	p.mx.Lock()
	defer p.mx.Unlock()
	for _, msg := range msgs {
		p.mp[msg.id] = msg
	}
}

func (p *msgMap) get(id ShipmentID) *msgShipment {
	p.mx.RLock()
	msg := p.mp[id]
	p.mx.RUnlock()
	return msg
}

func (p *msgMap) delete(id ShipmentID) {
	p.mx.Lock()
	delete(p.mp, id)
	p.mx.Unlock()
}

func (p *msgMap) deleteAll(ids []retries.RetryID) {
	if len(ids) == 0 {
		return
	}
	p.mx.Lock()
	for _, id := range ids {
		delete(p.mp, ShipmentID(id))
	}
	p.mx.Unlock()
}

func (p *msgMap) deleteMsgList(msgs []*msgShipment) {
	if len(msgs) == 0 {
		return
	}
	p.mx.Lock()
	defer p.mx.Unlock()
	for _, msg := range msgs {
		delete(p.mp, msg.id)
	}
}

func (p *msgMap) deleteAllAndShrink(ids []retries.RetryID) []retries.RetryID {
	if len(ids) == 0 {
		return nil
	}
	p.mx.Lock()
	j := 0
	for i, id := range ids {
		shid := ShipmentID(id)
		if _, ok := p.mp[shid]; ok {
			delete(p.mp, shid)
			continue
		}
		if i != j {
			ids[j] = id
		}
		j++
	}
	p.mx.Unlock()
	return ids[:j]
}
