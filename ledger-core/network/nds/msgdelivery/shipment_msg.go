package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
)

type ShipmentState uint8

const (
	_ ShipmentState = iota
	WaitAck
	BodyReady
	BodyRequested
	BodySending
	WaitBodyAck
	Done
	Rejected
	Expired
	Cancelled
)

type msgShipment struct {
	id      ShipmentID
	state   atomickit.Uint32 // ShipmentState
	expires uint32           // after this cycle

	returnID ShortShipmentID
	shipment Shipment

	peer *DeliveryPeer
}

func (p *msgShipment) isImmediateSend() bool {
	return true
}

func (p *msgShipment) isFireAndForget() bool {
	return p.shipment.Policies&FireAndForget != 0
}

func (p *msgShipment) getHeadRetryState() retries.RetryState {
	switch state := p.getState(); {
	case state >= Done:
		return retries.RemoveCompletely
	case p._isExpired():
		return retries.RemoveCompletely
	case !p.peer.isValid():
		return retries.RemoveCompletely
	case p.canSendHead():
		return retries.KeepRetrying
	default:
		return retries.StopRetrying
	}
}

func (p *msgShipment) canSendHead() bool {
	return p.getState() <= WaitAck
}

func (p *msgShipment) canSendBody() bool {
	state := p.getState()
	return state >= BodyRequested && state < Done
}

func (p *msgShipment) getState() ShipmentState {
	return ShipmentState(p.state.Load())
}

func (p *msgShipment) markAck() {
	newState := Done
	if p.shipment.Body != nil {
		newState = BodyReady
	}

	for {
		state := p.getState()
		if state <= WaitAck && !p.state.CompareAndSwap(uint32(state), uint32(newState)) {
			continue
		}
		return
	}
}

func (p *msgShipment) markBodyRq() bool {
	for {
		switch state := p.getState(); {
		case state >= Done:
			return false
		case state == BodySending:
			return false
		case state >= BodyRequested:
			return true
		case p.state.CompareAndSwap(uint32(state), uint32(BodyRequested)):
			return true
		}
	}
}

func (p *msgShipment) _markCompletion(complete ShipmentState) {
	if complete < Done {
		panic(throw.IllegalValue())
	}
	for {
		switch state := p.getState(); {
		case state >= Done:
			return
		case p.state.CompareAndSwap(uint32(state), uint32(complete)):
			// we may have to take additional measures to do cleanup shipment earlier
			return
		}
	}
}

func (p *msgShipment) markBodyAck() {
	p._markCompletion(Done)
}

func (p *msgShipment) markReject() {
	p._markCompletion(Rejected)
}

func (p *msgShipment) markExpired() {
	p._markCompletion(Expired)
}

func (p *msgShipment) markCancel() {
	p._markCompletion(Cancelled)
}

func (p *msgShipment) _isExpired() bool {
	if cycle, _ := p.peer.ctl.getPulseCycle(); cycle > p.expires {
		p.markExpired()
		return true
	}
	if p.shipment.Cancel.IsCancelled() {
		p.markExpired()
		return true
	}
	return false
}

func (p *msgShipment) sendHead() bool {
	state := p.getState()
	if state > WaitAck {
		return false
	}

	defer p.state.CompareAndSwap(0, uint32(WaitAck))
	p.peer.sendParcel(p, false, state == WaitAck)
	return true
}

func (p *msgShipment) sendBody() {
	isRepeated := false
	for {
		switch state := p.getState(); state {
		case BodyReady, BodyRequested, WaitBodyAck:
			isRepeated = state == WaitBodyAck
			if !p.state.CompareAndSwap(uint32(state), uint32(BodySending)) {
				continue
			}
		default:
			return
		}
		break
	}

	if p.shipment.Policies&largeBody != 0 {
		go func() {
			defer p.state.CompareAndSwap(uint32(BodySending), uint32(WaitBodyAck))
			p.peer.sendLargeParcel(p, isRepeated)
		}()
		return
	}

	defer p.state.CompareAndSwap(uint32(BodySending), uint32(WaitBodyAck))
	p.peer.sendParcel(p, true, isRepeated)
}
