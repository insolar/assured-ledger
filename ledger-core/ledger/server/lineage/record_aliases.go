// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)


const tRLifelineStart = RecordType(rms.TypeRLifelineStartPolymorthID)
const tRSidelineStart = RecordType(rms.TypeRSidelineStartPolymorthID)

const tRLineInboundRequest = RecordType(rms.TypeRLineInboundRequestPolymorthID)

const tRInboundRequest = RecordType(rms.TypeRInboundRequestPolymorthID)

const tROutboundRequest = RecordType(rms.TypeROutboundRequestPolymorthID)
const tROutRetryableRequest = RecordType(rms.TypeROutboundRetryableRequestPolymorthID)
const tROutRetryRequest = RecordType(rms.TypeROutboundRetryRequestPolymorthID)
const tROutboundResponse = RecordType(rms.TypeROutboundResponsePolymorthID)
const tRInboundResponse = RecordType(rms.TypeRInboundResponsePolymorthID)

const tRLineActivate = RecordType(rms.TypeRLineActivatePolymorthID)
const tRLineDeactivate = RecordType(rms.TypeRLineDeactivatePolymorthID)

const tRLineMemoryInit = RecordType(rms.TypeRLineMemoryInitPolymorthID)
const tRLineMemory = RecordType(rms.TypeRLineMemoryPolymorthID)
const tRLineMemoryReuse = RecordType(rms.TypeRLineMemoryReusePolymorthID)
const tRLineMemoryExpected = RecordType(rms.TypeRLineMemoryExpectedPolymorthID)
const tRLineMemoryProvided = RecordType(rms.TypeRLineMemoryProvidedPolymorthID)


const tRLineRecap = RecordType(rms.TypeRLineRecapPolymorthID)

func appendCopy(v []RecordType, u... RecordType) []RecordType {
	return append(append([]RecordType(nil), v...), u...)
}
