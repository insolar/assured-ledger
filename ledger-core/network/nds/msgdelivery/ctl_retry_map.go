package msgdelivery

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
)

type ttlMap struct {
	mutex sync.RWMutex
	ttl0  map[ShipmentID]*msgShipment
	ttl1  map[ShipmentID]*msgShipment
	ttlN  map[ShipmentID]*msgShipment
}

func (p *ttlMap) put(msg *msgShipment, currentCycle uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case msg.expires <= currentCycle:
		if p.ttl0 == nil {
			p.ttl0 = make(map[ShipmentID]*msgShipment)
		}
		p.ttl0[msg.id] = msg
	case msg.expires == currentCycle+1:
		if p.ttl1 == nil {
			p.ttl1 = make(map[ShipmentID]*msgShipment)
		}
		p.ttl1[msg.id] = msg
	default:
		if p.ttlN == nil {
			p.ttlN = make(map[ShipmentID]*msgShipment)
		}
		p.ttlN[msg.id] = msg
	}
}

func (p *ttlMap) get(id ShipmentID) *msgShipment {
	p.mutex.RLock()
	msg := p.ttl0[id]
	if msg == nil {
		if msg = p.ttl1[id]; msg == nil {
			msg = p.ttlN[id]
		}
	}
	p.mutex.RUnlock()
	return msg
}

func (p *ttlMap) deleteAll(ids []retries.RetryID) {
	if len(ids) == 0 {
		return
	}
	p.mutex.Lock()
	for _, id := range ids {
		sid := ShipmentID(id)
		delete(p.ttl0, sid)
		delete(p.ttl1, sid)
		delete(p.ttlN, sid)
	}
	p.mutex.Unlock()
}

func (p *ttlMap) nextTTLCycle(currentCycle uint32) {
	p.mutex.Lock()

	p.ttl0 = p.ttl1
	p.ttl1 = nil
	if n := len(p.ttl0); n > 64 {
		p.ttl1 = make(map[ShipmentID]*msgShipment, n>>1)
	}

	if len(p.ttlN) == 0 {
		p.ttlN = nil
		p.mutex.Unlock()
		return
	}

	// this is to return from nextTTLCycle within a constant time
	go p._nextTTLCycle(currentCycle)
}

func (p *ttlMap) _nextTTLCycle(currentCycle uint32) {
	defer p.mutex.Unlock()

	for id, msg := range p.ttlN {
		switch {
		case msg.expires <= currentCycle:
			if p.ttl0 == nil {
				p.ttl0 = make(map[ShipmentID]*msgShipment)
			}
			p.ttl0[id] = msg
		case msg.expires == currentCycle+1:
			if p.ttl1 == nil {
				p.ttl1 = make(map[ShipmentID]*msgShipment)
			}
			p.ttl1[id] = msg
		default:
			continue
		}
		delete(p.ttlN, id)
	}
}
