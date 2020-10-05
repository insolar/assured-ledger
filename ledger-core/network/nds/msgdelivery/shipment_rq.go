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

func (v rqShipment) isEmpty() bool {
	return v.peer == nil
}

func (v rqShipment) isExpired() bool {
	if cycle, _ := v.peer.ctl.getPulseCycle(); cycle > v.expires {
		return true
	}
	if v.request.Cancel.IsCancelled() {
		return true
	}
	return false
}

func (v rqShipment) requestRejectedFn() func() {
	fn := v.request.ReceiveFn
	if fn == nil {
		return nil
	}

	retAddr := ReturnAddress{
		returnTo: v.peer.peer.GetLocalUID(),
		returnID: v.id.ShortID(),
		expires:  v.expires,
	}
	return func() {
		if err := fn(retAddr, false, nil); err != nil {
			v.peer.ctl.reportError(err)
		}
	}
}

func (v rqShipment) isValid() bool {
	return v.peer.isValid() && !v.isExpired()
}
