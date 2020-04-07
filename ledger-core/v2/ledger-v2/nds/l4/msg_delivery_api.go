// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

type MessageDelivery interface {
	ShipTo(to DeliveryAddress, parcel DeliveryParcel, needsTag bool) (*DeliveryTag, error)
	ShipReturn(to ReturnAddress, parcel DeliveryParcel, needsTag bool) (*DeliveryTag, error)
}

type DeliveryTag struct {
}

type DeliveryPolicies uint8

const (
	// FireAndForget indicates that this delivery doesn't need ACK. Can only be applied to head-only parcels
	FireAndForget DeliveryPolicies = 1 << iota
	// ExpectedParcel indicates that receiver expects this parcel, and the parcel should be delivered as head+body
	ExpectedParcel
)

//type TransportResult uint8
//
//const (
//	TransportUnreachable TransportResult = iota
//	TransportSent
//	//TransportDelivered
//)
