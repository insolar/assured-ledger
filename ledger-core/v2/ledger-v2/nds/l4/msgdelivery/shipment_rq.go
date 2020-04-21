// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

type rqShipment struct {
	id      ShipmentID
	expires uint32
	peer    *DeliveryPeer
	request ShipmentRequest
}

func (p rqShipment) isExpired() bool {
	if cycle, _ := p.peer.ctl.getPulseCycle(); cycle > p.expires {
		return true
	}
	if p.request.Cancel.IsCancelled() {
		return true
	}
	return false
}

func (p rqShipment) requestRejected() {
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
