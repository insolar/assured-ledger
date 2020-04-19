// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type bodyRequester struct {
	mutex    sync.Mutex
	requests map[ShipmentID]rqShipment
}

type rqShipment struct {
	id      ShipmentID
	expires uint32
	peer    *DeliveryPeer
	request ShipmentRequest
}

func (p *rqShipment) isExpired() bool {
	panic(throw.NotImplemented()) // TODO
}

func (p *rqShipment) callbackRejected() {
	fn := p.request.ReceiveFn
	if fn == nil {
		return
	}

	retAddr := ReturnAddress{
		returnTo: p.peer.peer.GetLocalUID(),
		returnID: p.id.ShortID(),
		expires:  p.expires,
	}
	if err := fn(retAddr, false, nil); err != nil {
		p.peer.ctl.reportError(err)
	}
}

// TODO ttl and retries

func (p *bodyRequester) Add(rq rqShipment) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ok := p.requests[rq.id]; ok {
		return throw.FailHere("duplicate request")
	}
	p.requests[rq.id] = rq
	go rq.peer.sendBodyRq(rq.id.ShortID())
	return nil
}

func (p *bodyRequester) Remove(peer *DeliveryPeer, id ShortShipmentID) (rqShipment, bool) {
	return p.RemoveByID(AsShipmentID(uint32(peer.peerID), id))
}

func (p *bodyRequester) RemoveByID(sid ShipmentID) (rqShipment, bool) {
	p.mutex.Lock()
	rq, ok := p.requests[sid]
	if ok {
		delete(p.requests, sid)
	}
	p.mutex.Unlock()

	if !ok || rq.request.Cancel.IsCancelled() {
		return rqShipment{}, false
	}
	return rq, ok
}

func (p *bodyRequester) suspendRetry(sid ShipmentID) func() {
	// TODO
	return nil
}
