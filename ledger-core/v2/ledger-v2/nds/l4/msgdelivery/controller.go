// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
)

type ProtocolController struct {
	factory  apinetwork.DeserializationFactory
	receiver PacketReceiver

	lastOutgoingShId atomickit.Uint64 // ShipmentID
	currPulseShId    ShipmentID
	prevPulseShId    ShipmentID
}

func (p *ProtocolController) reportError(err error) {

}

func (p *ProtocolController) receiveState(payload *StatePacket) error {

}

func (p *ProtocolController) receiveComplete(payload *ParcelPacket) error {

}

func (p *ProtocolController) receiveBody(payload *ParcelPacket) error {

}

func (p *ProtocolController) receiveHead(payload *ParcelPacket) error {

}

func (p *ProtocolController) Ship(payload *ParcelPacket) error {
	// check valid
	//payload.
}

/**********************************/

type DeliveryPeer struct {
	nextShid atomickit.Uint32 // ShortShipmentId
	outbound *OutboundPeer
}

func (p *DeliveryPeer) NextShipmentId() ShipmentID {

}

type OutboundPeer struct {
	*DeliveryPeer
	isDead atomickit.OnceFlag
}

func (p *OutboundPeer) isValid() bool {
	return !p.isDead.IsSet()
}

func (p *OutboundPeer) invalidate() bool {
	return p.isDead.DoSet(func() {
		p.DeliveryPeer = nil
	})
}
