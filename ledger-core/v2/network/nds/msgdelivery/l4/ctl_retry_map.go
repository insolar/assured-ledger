// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/nds/msgdelivery/retries"
)

type msgMap struct {
	mx sync.RWMutex
	mp map[ShipmentID]*msgShipment
}

func (p *msgMap) put(msg *msgShipment) {
	p.mx.Lock()
	if p.mp == nil {
		p.mp = make(map[ShipmentID]*msgShipment)
	}
	p.mp[msg.id] = msg
	p.mx.Unlock()
}

//func (p *msgMap) putAll(msgs []*msgShipment) {
//	if len(msgs) == 0 {
//		return
//	}
//	p.mx.Lock()
//	defer p.mx.Unlock()
//	for _, msg := range msgs {
//		p.mp[msg.id] = msg
//	}
//}

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

//func (p *msgMap) deleteMsgList(msgs []*msgShipment) {
//	if len(msgs) == 0 {
//		return
//	}
//	p.mx.Lock()
//	defer p.mx.Unlock()
//	for _, msg := range msgs {
//		delete(p.mp, msg.id)
//	}
//}

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
