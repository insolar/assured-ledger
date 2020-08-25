// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

const (
	TypeRLifelineStartPolymorphID            = TypeROutboundRequestPolymorphID + 1
	TypeRSidelineStartPolymorphID            = TypeROutboundRequestPolymorphID + 2
	TypeROutboundRetryableRequestPolymorphID = TypeROutboundRequestPolymorphID + 3
	TypeROutboundRetryRequestPolymorphID     = TypeROutboundRequestPolymorphID + 4

	TypeRLineMemoryInitPolymorphID     = TypeRLineMemoryPolymorphID + 1
	TypeRLineMemoryProvidedPolymorphID = TypeRLineMemoryPolymorphID + 2
)

type (
	RLifelineStart            = ROutboundRequest
	RSidelineStart            = ROutboundRequest
	ROutboundRetryableRequest = ROutboundRequest
	ROutboundRetryRequest     = ROutboundRequest

	RLineMemoryInit     = RLineMemory
	RLineMemoryProvided = RLineMemory
)

func init() {
	RegisterRecordType(TypeRLifelineStartPolymorphID, "", (*RLifelineStart)(nil))
	RegisterRecordType(TypeRSidelineStartPolymorphID, "", (*RSidelineStart)(nil))
	RegisterRecordType(TypeROutboundRetryableRequestPolymorphID, "", (*ROutboundRetryableRequest)(nil))
	RegisterRecordType(TypeROutboundRetryRequestPolymorphID, "", (*ROutboundRetryRequest)(nil))

	RegisterRecordType(TypeRLineMemoryInitPolymorphID, "", (*RLineMemoryInit)(nil))
	RegisterRecordType(TypeRLineMemoryProvidedPolymorphID, "", (*RLineMemoryProvided)(nil))
}
