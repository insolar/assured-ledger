// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

type msgShipment struct {
	id       ShipmentID
	returnId ShortShipmentId
	shipment Shipment

	peer *OutboundPeer
}

func (p *msgShipment) allowsBatching() bool {
	return true
}

func (p *msgShipment) getBatchWeight() int {

}

func (p *msgShipment) isDone() bool {
	return p._isDone() || !p.peer.isValid()
}

func (p *msgShipment) _isDone() bool {

}

func (p *msgShipment) markReceived() {

}
