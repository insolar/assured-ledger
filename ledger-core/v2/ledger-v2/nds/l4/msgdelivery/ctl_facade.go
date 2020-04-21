// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

var _ Service = facade{}

type facade struct {
	ctl *Controller
}

func (v facade) ShipTo(to DeliveryAddress, shipment Shipment) error {
	return v.ctl.shipTo(to, shipment)
}

func (v facade) ShipReturn(to ReturnAddress, shipment Shipment) error {
	return v.ctl.shipReturn(to, shipment)
}

func (v facade) PullBody(from ReturnAddress, shipmentRq ShipmentRequest) error {
	return v.ctl.pullBody(from, shipmentRq)
}

func (v facade) RejectBody(from ReturnAddress) error {
	return v.ctl.rejectBody(from)
}
