// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
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
)

type msgShipment struct {
	id      ShipmentID
	state   atomickit.Uint32 // ShipmentState
	expires uint32           // after this cycle

	returnId ShortShipmentID
	shipment Shipment

	peer *DeliveryPeer
}

func (p *msgShipment) isImmediateSend() bool {
	return false
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
	case state <= WaitAck:
		return retries.KeepRetrying
	default:
		return retries.StopRetrying
	}
}

func (p *msgShipment) getBodyRetryState() retries.RetryState {
	switch state := p.getState(); {
	case state >= Done:
		return retries.RemoveCompletely
	case p._isExpired():
		return retries.RemoveCompletely
	case !p.peer.isValid():
		return retries.RemoveCompletely
	case state >= BodyRequested && state < Done:
		return retries.KeepRetrying
	default:
		return retries.StopRetrying
	}
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

func (p *msgShipment) markBodyAck() {
	for {
		state := p.getState()
		if state < Done && !p.state.CompareAndSwap(uint32(state), uint32(Done)) {
			continue
		}
		return
	}
}

func (p *msgShipment) markReject() {
	for {
		state := p.getState()
		if state < Done && !p.state.CompareAndSwap(uint32(state), uint32(Rejected)) {
			continue
		}
		return
	}
}

func (p *msgShipment) markBodyRq() bool {
	for {
		switch state := p.getState(); {
		case state >= BodyRequested:
			return false
		case p.state.CompareAndSwap(uint32(state), uint32(BodyRequested)):
			return true
		}
	}
}

func (p *msgShipment) markExpired() {
	for {
		state := p.getState()
		if state < Done && !p.state.CompareAndSwap(uint32(state), uint32(Expired)) {
			continue
		}
		return
	}
}

func (p *msgShipment) _isExpired() bool {
	cycle, _ := p.peer.ctl.getPulseCycle()
	if cycle > p.expires {
		p.markExpired()
		return true
	}
	return false
}

func (p *msgShipment) sendHead(isRepeated bool) {
	if p.getState() > WaitAck {
		return
	}

	defer p.state.CompareAndSwap(0, uint32(WaitAck))
	p.peer.sendParcel(p, false, isRepeated)
}

func (p *msgShipment) sendBody(isRepeated bool) {
	for {
		switch state := p.getState(); state {
		case BodyReady, BodyRequested, WaitBodyAck:
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
