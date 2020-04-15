// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import "github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/l4/msgdelivery/retries"

type msgShipment struct {
	id       ShipmentID
	returnId ShortShipmentID
	shipment Shipment

	peer *DeliveryPeer
}

func (p *msgShipment) allowsBatching() bool {
	return true
}

func (p *msgShipment) getBatchWeight() int {

}

func (p *msgShipment) checkState() retries.RetryState {
	return p._isDone() || !p.peer.isValid()
}

func (p *msgShipment) isBodyDone() bool {
	return p._isDone() || !p.peer.isValid()
}

func (p *msgShipment) _isDone() bool {

}

func (p *msgShipment) markAck() {

}

func (p *msgShipment) markBodyAck() {

}

func (p *msgShipment) markReject() {

}

func (p *msgShipment) send(retry bool) {

}

func (p *msgShipment) markBodyRq() bool {

}
