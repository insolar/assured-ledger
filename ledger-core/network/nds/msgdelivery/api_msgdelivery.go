package msgdelivery

type Service interface {
	ShipTo(to DeliveryAddress, shipment Shipment) error
	ShipReturn(to ReturnAddress, shipment Shipment) error
	PullBody(from ReturnAddress, shipmentRq ShipmentRequest) error
	RejectBody(from ReturnAddress) error
}

type DeliveryPolicies uint8

const (
	// FireAndForget indicates that this delivery doesn't need ACK. Can only be applied to head-only parcels
	FireAndForget DeliveryPolicies = 1 << iota
	// ExpectedParcel indicates that receiver expects this shipment, and the shipment should be delivered as head+body
	ExpectedParcel

	ExactPulse

	largeBody
)
