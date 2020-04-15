// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"math"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func NewRetryController() *RetryController {

}

type RetryController struct {
	sendPeriod time.Duration

	retries retries.StagedController

	ttl0 ttlAge
	ttl1 ttlAge
	ttlN ttlAge
}

func (p *RetryController) Send(to *DeliveryPeer, msg *msgShipment) {
	id := to.NextShipmentId()
	msg.id = id
	msg.peer = to.outbound

	switch msg.shipment.TTL {
	case 0:
		p.ttl0.put(msg)
	case 1:
		p.ttl1.put(msg)
	default:
		p.ttlN.put(msg)
	}

	wt := math.MaxInt64
	if msg.allowsBatching() {
		wt = msg.getBatchWeight()
	}

	if id := retries.RetryID(msg.id); !p.retries.Add(id, wt, p) {
		p._sendMsg(msg)
		p.retries.AddForRetry(id)
	}
}

func (p *RetryController) Retry(ids []retries.RetryID, repeatFn func([]retries.RetryID)) {
	j := 0
	for i, id := range ids {
		msg := p.get(ShipmentID(id))
		if msg == nil || msg.isDone() {
			continue
		}
		p._sendMsg(msg)
		if j != i {
			ids[j] = id
		}
		j++
	}

	if j > 0 {
		repeatFn(ids[:j])
	}
}

func (p *RetryController) Verify(id retries.RetryID) bool {
	if msg := p.get(ShipmentID(id)); msg != nil {
		return msg.isDone()
	}
	return false
}

func (p *RetryController) get(shid ShipmentID) *msgShipment {
	if msg := p.ttl0.get(shid); msg != nil {
		return msg
	}
	if msg := p.ttl1.get(shid); msg != nil {
		return msg
	}
	if msg := p.ttlN.get(shid); msg != nil {
		return msg
	}
	return nil
}

func (p *RetryController) Remove(ids []retries.RetryID) {
	if len(ids) == 0 {
		return
	}
	if ids = p.ttl0.deleteAllAndShrink(ids); len(ids) == 0 {
		return
	}
	if ids = p.ttl1.deleteAllAndShrink(ids); len(ids) == 0 {
		return
	}
	p.ttlN.deleteAll(ids)
}

func (p *RetryController) NextCycle() {
	p.retries.NextCycle(p)
}

func (p *RetryController) NextTTLCycle() {
	p.ttl1.moveTo(&p.ttl0)
	p.ttlN.flushTo(&p.ttl0, 1)
}

func (p *RetryController) _sendMsg(in *msgShipment) bool {
	// send
}

/**********************************/

type ttlAge struct {
	mx sync.RWMutex
	mp map[ShipmentID]*msgShipment
}

func (p *ttlAge) put(msg *msgShipment) {
	p.mx.Lock()
	p.mp[msg.id] = msg
	p.mx.Unlock()
}

func (p *ttlAge) get(id ShipmentID) *msgShipment {
	p.mx.RLock()
	msg := p.mp[id]
	p.mx.RUnlock()
	return msg
}

func (p *ttlAge) delete(id ShipmentID) {
	p.mx.Lock()
	delete(p.mp, id)
	p.mx.Unlock()
}

func (p *ttlAge) deleteAll(ids []retries.RetryID) {
	p.mx.Lock()
	for _, id := range ids {
		delete(p.mp, ShipmentID(id))
	}
	p.mx.Unlock()
}

func (p *ttlAge) markReceived(id ShipmentID) bool {
	p.mx.RLock()
	msg := p.mp[id]
	p.mx.RUnlock()
	if msg == nil {
		return false
	}
	msg.markReceived()
	return true
}

func (p *ttlAge) deleteAllAndShrink(ids []retries.RetryID) []retries.RetryID {
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

func (p *ttlAge) flushTo(ttl0 *ttlAge, lowCut uint8) {
	p.mx.Lock()
	defer p.mx.Unlock()

	ttl0.mx.Lock()
	defer ttl0.mx.Unlock()

	for id, msg := range p.mp {
		if msg.shipment.TTL > lowCut {
			msg.shipment.TTL--
			continue
		}
		delete(p.mp, id)
		ttl0.mp[id] = msg
	}
}

func (p *ttlAge) moveTo(ttl0 *ttlAge) {
	if ttl0 == nil {
		panic(throw.IllegalValue())
	}

	p.mx.Lock()
	if n := len(p.mp); n > 0 {
		ttl0.mx.Lock()
		ttl0.mp = p.mp
		ttl0.mx.Unlock()

		p.mp = make(map[ShipmentID]*msgShipment, n)
		p.mx.Unlock()
		return
	}
	p.mp = nil
	p.mx.Unlock()

	ttl0.flushAll()
}

func (p *ttlAge) flushAll() {
	p.mx.Lock()
	defer p.mx.Unlock()

	switch n := len(p.mp); {
	case n == 0:
		//
	case n > 32:
		p.mp = make(map[ShipmentID]*msgShipment, n)
	default:
		for id, _ := range p.mp {
			delete(p.mp, id)
		}
	}
}
