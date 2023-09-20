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
