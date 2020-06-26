// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

const (
	TypeRLifelineStartPolymorthID = TypeROutboundRequestPolymorthID +1
	TypeRSidelineStartPolymorthID = TypeROutboundRequestPolymorthID +2
	TypeROutboundRetryableRequestPolymorthID = TypeROutboundRequestPolymorthID +3
	TypeROutboundRetryRequestPolymorthID = TypeROutboundRequestPolymorthID +4

	TypeRLineMemoryInitPolymorthID = TypeRLineMemoryPolymorthID +1
	TypeRLineMemoryProvidedPolymorthID = TypeRLineMemoryPolymorthID +2
)

func init() {
	RegisterRecordType(TypeRLifelineStartPolymorthID, "", (*ROutboundRequest)(nil))
	RegisterRecordType(TypeRSidelineStartPolymorthID, "", (*ROutboundRequest)(nil))
	RegisterRecordType(TypeRLineMemoryInitPolymorthID, "", (*RLineMemory)(nil))
}
