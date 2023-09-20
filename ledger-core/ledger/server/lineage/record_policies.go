package lineage

type RecordPolicyProviderFunc = func (recordType RecordType) RecordPolicy

func GetRecordPolicy(recordType RecordType) RecordPolicy {
	if recordType == 0 || int(recordType) >= len(policies) {
		return RecordPolicy{}
	}
	return policies[recordType]
}

const maxRecordType = 1000

var setLineStart = []RecordType{tRLifelineStart, tRSidelineStart}
var setUnblockedLine = setOf(tRLineActivate, tRLineMemory, tRLineMemoryReuse, tRLineMemoryExpected, tRLineMemoryProvided)
var setUnblockedInbound = []RecordType{tRLineInboundRequest, tRInboundRequest, tROutboundResponse}
var setDirectActivate = setOf(appendCopy(setLineStart, tRLineMemoryInit)...)

var policies = []RecordPolicy{
	tRLifelineStart: 		{ PolicyFlags: LineStart|ReasonRequired|PayloadAllowed},
	tRSidelineStart: 		{ PolicyFlags: LineStart| BranchedStart |ReasonRequired|PayloadAllowed},

	tRLineInboundRequest:	{
		PolicyFlags: FilamentStart|ReasonRequired|PayloadAllowed,
		CanFollow:   setUnblockedLine},

	tRInboundRequest: 	 	{
		PolicyFlags: FilamentStart| BranchedStart |ReasonRequired|PayloadAllowed,
		CanFollow:   setUnblockedLine},

	tRInboundResponse:		{
		PolicyFlags: FilamentEnd| MustBeBranch|PayloadAllowed,
		CanFollow:   setOf(appendCopy(setUnblockedInbound, tROutboundRequest, tROutRetryableRequest, tROutRetryRequest)...)},

	tROutboundRequest: 		{
		PolicyFlags: MustBeBranch|PayloadAllowed,
		CanFollow:   setOf(setUnblockedInbound...)},

	tROutRetryableRequest:	{
		PolicyFlags: MustBeBranch|PayloadAllowed,
		CanFollow:   setOf(setUnblockedInbound...)},

	tROutRetryRequest:		{
		PolicyFlags: NextPulseOnly|OnlyHash,
		CanFollow:   setOf(tROutRetryableRequest, tROutRetryRequest), RedirectTo: setOf(tROutRetryableRequest)},

	tROutboundResponse: 	{
		PolicyFlags: PayloadAllowed,
		CanFollow:   setOf(tROutboundRequest, tROutRetryableRequest, tROutRetryRequest)},

	tRLineActivate: 		{ // NB! Special behavior. See RecordPolicy.CheckRejoinRef
		PolicyFlags: SideEffect,
		CanFollow:   setOf(appendCopy(setLineStart, tRLineMemoryInit, tRLineMemory)...)},

	tRLineDeactivate: 		{ // NB! Special behavior. See RecordPolicy.CheckPrevRef
		PolicyFlags: FilamentEnd|SideEffect,
		CanFollow:   setUnblockedLine},

	tRLineMemoryInit: 		{
		PolicyFlags: PayloadAllowed,
		CanFollow:   setOf(setLineStart...)},

	tRLineMemory: 			{
		PolicyFlags: SideEffect|PayloadAllowed,
		CanFollow:   setOf(tRLineInboundRequest, tRLineMemoryExpected)},

	tRLineMemoryReuse: 		{
		PolicyFlags: SideEffect,
		CanFollow:   setOf(tRLineInboundRequest), RedirectTo: setOf(tRLineMemory, tRLineMemoryProvided) },

	tRLineMemoryExpected: 	{ // NB! Special behavior. See RecordPolicy.CheckPrevRef
		PolicyFlags: SideEffect|OnlyHash|BlockNextPulse,
		CanFollow:   setOf(tRLineInboundRequest)},

	tRLineMemoryProvided: 	{ // NB! Special behavior. See RecordPolicy.CheckPrevRef
		PolicyFlags: PayloadAllowed,
		CanFollow:   setOf(tRLineMemoryExpected)},

	tRLineRecap:	{
		PolicyFlags: Recap|NextPulseOnly },
}
