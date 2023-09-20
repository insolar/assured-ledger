package rms

const (
	TypeRLineInboundRequestPolymorphID       = TypeROutboundRequestPolymorphID + 1
	TypeRInboundRequestPolymorphID           = TypeROutboundRequestPolymorphID + 2
	TypeRLifelineStartPolymorphID            = TypeROutboundRequestPolymorphID + 3
	TypeRSidelineStartPolymorphID            = TypeROutboundRequestPolymorphID + 4
	TypeROutboundRetryableRequestPolymorphID = TypeROutboundRequestPolymorphID + 5
	TypeROutboundRetryRequestPolymorphID     = TypeROutboundRequestPolymorphID + 6

	TypeRLineMemoryInitPolymorphID     = TypeRLineMemoryPolymorphID + 1
	TypeRLineMemoryProvidedPolymorphID = TypeRLineMemoryPolymorphID + 2
)

type (
	RLineInboundRequest       = ROutboundRequest
	RInboundRequest           = ROutboundRequest
	RLifelineStart            = ROutboundRequest
	RSidelineStart            = ROutboundRequest
	ROutboundRetryableRequest = ROutboundRequest
	ROutboundRetryRequest     = ROutboundRequest

	RLineMemoryInit     = RLineMemory
	RLineMemoryProvided = RLineMemory
)

func init() {
	RegisterRecordType(TypeRLineInboundRequestPolymorphID, "", (*RLineInboundRequest)(nil))
	RegisterRecordType(TypeRInboundRequestPolymorphID, "", (*RInboundRequest)(nil))
	RegisterRecordType(TypeRLifelineStartPolymorphID, "", (*RLifelineStart)(nil))
	RegisterRecordType(TypeRSidelineStartPolymorphID, "", (*RSidelineStart)(nil))
	RegisterRecordType(TypeROutboundRetryableRequestPolymorphID, "", (*ROutboundRetryableRequest)(nil))
	RegisterRecordType(TypeROutboundRetryRequestPolymorphID, "", (*ROutboundRetryRequest)(nil))

	RegisterRecordType(TypeRLineMemoryInitPolymorphID, "", (*RLineMemoryInit)(nil))
	RegisterRecordType(TypeRLineMemoryProvidedPolymorphID, "", (*RLineMemoryProvided)(nil))
}
