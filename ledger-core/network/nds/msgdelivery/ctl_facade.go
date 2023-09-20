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
