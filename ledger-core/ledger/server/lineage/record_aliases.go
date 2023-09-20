package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

const (
	tRLifelineStart = RecordType(rms.TypeRLifelineStartPolymorphID)
	tRSidelineStart = RecordType(rms.TypeRSidelineStartPolymorphID)

	tRLineInboundRequest = RecordType(rms.TypeRLineInboundRequestPolymorphID)
	tRInboundRequest     = RecordType(rms.TypeRInboundRequestPolymorphID)
	tRInboundResponse    = RecordType(rms.TypeRInboundResponsePolymorphID)

	tROutboundRequest     = RecordType(rms.TypeROutboundRequestPolymorphID)
	tROutRetryableRequest = RecordType(rms.TypeROutboundRetryableRequestPolymorphID)
	tROutRetryRequest     = RecordType(rms.TypeROutboundRetryRequestPolymorphID)
	tROutboundResponse    = RecordType(rms.TypeROutboundResponsePolymorphID)

	tRLineActivate   = RecordType(rms.TypeRLineActivatePolymorphID)
	tRLineDeactivate = RecordType(rms.TypeRLineDeactivatePolymorphID)

	tRLineMemory         = RecordType(rms.TypeRLineMemoryPolymorphID)
	tRLineMemoryInit     = RecordType(rms.TypeRLineMemoryInitPolymorphID)
	tRLineMemoryReuse    = RecordType(rms.TypeRLineMemoryReusePolymorphID)
	tRLineMemoryExpected = RecordType(rms.TypeRLineMemoryExpectedPolymorphID)
	tRLineMemoryProvided = RecordType(rms.TypeRLineMemoryProvidedPolymorphID)

	tRLineRecap = RecordType(rms.TypeRLineRecapPolymorphID)
)

func appendCopy(v []RecordType, u ...RecordType) []RecordType {
	return append(append([]RecordType(nil), v...), u...)
}
