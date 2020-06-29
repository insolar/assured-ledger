// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	tRLifelineStart: 		{ FieldPolicy: LineStart|ReasonRequired },
	tRSidelineStart: 		{ FieldPolicy: LineStart|Branched|ReasonRequired },

	tRLineInboundRequest:	{
		FieldPolicy: FilamentStart|ReasonRequired,
		CanFollow: setUnblockedLine},

	tRInboundRequest: 	 	{
		FieldPolicy: FilamentStart|Branched|ReasonRequired,
		CanFollow: setUnblockedLine},

	tRInboundResponse:		{
		FieldPolicy: FilamentEnd,
		CanFollow: setOf(appendCopy(setUnblockedInbound, tROutboundRequest, tROutRetryableRequest, tROutRetryRequest)...)},

	tROutboundRequest: 		{
		FieldPolicy: 0,
		CanFollow: setOf(setUnblockedInbound...)},

	tROutRetryableRequest:	{
		FieldPolicy: 0,
		CanFollow: setOf(setUnblockedInbound...)},

	tROutRetryRequest:		{
		FieldPolicy: NextPulseOnly|OnlyHash,
		CanFollow: setOf(tROutRetryableRequest, tROutRetryRequest), RedirectTo: setOf(tROutRetryableRequest)},

	tROutboundResponse: 	{
		FieldPolicy: 0,
		CanFollow: setOf(tROutboundRequest, tROutRetryableRequest, tROutRetryRequest)},

	tRLineActivate: 		{
		FieldPolicy: SideEffect|Unblocked,
		CanFollow: setOf(appendCopy(setLineStart, tRLineMemoryInit, tRLineMemory)...)},

	tRLineDeactivate: 		{
		FieldPolicy: FilamentEnd|SideEffect,
		CanFollow: setUnblockedLine},

	tRLineMemoryInit: 		{
		FieldPolicy: 0,
		CanFollow: setOf(setLineStart...)},

	tRLineMemory: 			{
		FieldPolicy: SideEffect|Unblocked,
		CanFollow: setOf(tRLineInboundRequest, tRLineMemoryExpected)},

	tRLineMemoryReuse: 		{
		FieldPolicy: SideEffect|Unblocked,
		CanFollow: setOf(tRLineInboundRequest), RedirectTo: setOf(tRLineMemory, tRLineMemoryProvided) },

	tRLineMemoryExpected: 	{
		FieldPolicy: SideEffect|OnlyHash,
		CanFollow: setOf(tRLineInboundRequest)},

	tRLineMemoryProvided: 	{
		FieldPolicy: Unblocked,
		CanFollow: setOf(tRLineMemoryExpected)},

	tRLineRecap:	{
		FieldPolicy: Recap|NextPulseOnly },
}
