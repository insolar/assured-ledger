// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

const (
	TypeRLifelineStartPolymorthID = TypeROutboundRequestPolymorthID +1
	TypeRSidelineStartPolymorthID = TypeROutboundRequestPolymorthID +2
<<<<<<< HEAD
	TypeROutboundRetryableRequestPolymorthID = TypeROutboundRequestPolymorthID +3
	TypeROutboundRetryRequestPolymorthID = TypeROutboundRequestPolymorthID +4

	TypeRLineMemoryInitPolymorthID = TypeRLineMemoryPolymorthID +1
	TypeRLineMemoryProvidedPolymorthID = TypeRLineMemoryPolymorthID +2
)

type (
	RLifelineStart = ROutboundRequest
	RSidelineStart = ROutboundRequest
	ROutboundRetryableRequest = ROutboundRequest
	ROutboundRetryRequest = ROutboundRequest

	RLineMemoryInit = RLineMemory
	RLineMemoryProvided = RLineMemory
)

func init() {
	RegisterRecordType(TypeRLifelineStartPolymorthID, "", (*RLifelineStart)(nil))
	RegisterRecordType(TypeRSidelineStartPolymorthID, "", (*RSidelineStart)(nil))
	RegisterRecordType(TypeROutboundRetryableRequestPolymorthID, "", (*ROutboundRetryableRequest)(nil))
	RegisterRecordType(TypeROutboundRetryRequestPolymorthID, "", (*ROutboundRetryRequest)(nil))

	RegisterRecordType(TypeRLineMemoryInitPolymorthID, "", (*RLineMemoryInit)(nil))
	RegisterRecordType(TypeRLineMemoryProvidedPolymorthID, "", (*RLineMemoryProvided)(nil))
=======
	TypeRLineMemoryInitPolymorthID = TypeRLineMemoryPolymorthID +1
)

func init() {
	RegisterRecordType(TypeRLifelineStartPolymorthID, "", (*ROutboundRequest)(nil))
	RegisterRecordType(TypeRSidelineStartPolymorthID, "", (*ROutboundRequest)(nil))
	RegisterRecordType(TypeRLineMemoryInitPolymorthID, "", (*RLineMemory)(nil))
>>>>>>> Further work
}
