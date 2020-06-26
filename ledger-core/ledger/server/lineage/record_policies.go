// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

<<<<<<< HEAD
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
	tRLifelineStart: 		{ PolicyFlags: LineStart|ReasonRequired },
	tRSidelineStart: 		{ PolicyFlags: LineStart| BranchedStart |ReasonRequired },

	tRLineInboundRequest:	{
		PolicyFlags: FilamentStart|ReasonRequired,
		CanFollow:   setUnblockedLine},

	tRInboundRequest: 	 	{
		PolicyFlags: FilamentStart| BranchedStart |ReasonRequired,
		CanFollow:   setUnblockedLine},

	tRInboundResponse:		{
		PolicyFlags: FilamentEnd| MustBeBranch,
		CanFollow:   setOf(appendCopy(setUnblockedInbound, tROutboundRequest, tROutRetryableRequest, tROutRetryRequest)...)},

	tROutboundRequest: 		{
		PolicyFlags: MustBeBranch,
		CanFollow:   setOf(setUnblockedInbound...)},

	tROutRetryableRequest:	{
		PolicyFlags: MustBeBranch,
		CanFollow:   setOf(setUnblockedInbound...)},

	tROutRetryRequest:		{
		PolicyFlags: NextPulseOnly|OnlyHash,
		CanFollow:   setOf(tROutRetryableRequest, tROutRetryRequest), RedirectTo: setOf(tROutRetryableRequest)},

	tROutboundResponse: 	{
		PolicyFlags: 0,
		CanFollow:   setOf(tROutboundRequest, tROutRetryableRequest, tROutRetryRequest)},

	tRLineActivate: 		{ // NB! Special behavior. See RecordPolicy.CheckRejoinRef
		PolicyFlags: SideEffect,
		CanFollow:   setOf(appendCopy(setLineStart, tRLineMemoryInit, tRLineMemory)...)},

	tRLineDeactivate: 		{ // NB! Special behavior. See RecordPolicy.CheckPrevRef
		PolicyFlags: FilamentEnd|SideEffect,
		CanFollow:   setUnblockedLine},

	tRLineMemoryInit: 		{
		PolicyFlags: 0,
		CanFollow:   setOf(setLineStart...)},

	tRLineMemory: 			{
		PolicyFlags: SideEffect,
		CanFollow:   setOf(tRLineInboundRequest, tRLineMemoryExpected)},

	tRLineMemoryReuse: 		{
		PolicyFlags: SideEffect,
		CanFollow:   setOf(tRLineInboundRequest), RedirectTo: setOf(tRLineMemory, tRLineMemoryProvided) },

	tRLineMemoryExpected: 	{ // NB! Special behavior. See RecordPolicy.CheckPrevRef
		PolicyFlags: SideEffect|OnlyHash|BlockNextPulse,
		CanFollow:   setOf(tRLineInboundRequest)},

	tRLineMemoryProvided: 	{ // NB! Special behavior. See RecordPolicy.CheckPrevRef
		PolicyFlags: 0,
		CanFollow:   setOf(tRLineMemoryExpected)},

	tRLineRecap:	{
		PolicyFlags: Recap|NextPulseOnly },
=======
import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type RecordPolicy struct {
	FieldPolicy
	MemberOf PrecedenceGroup
	CanFollow uint32
}

type FieldPolicy uint16

const (
	AllowEmptyRoot FieldPolicy = 1<<iota
	AllowEmptyPrev
	AllowEmptyReason
	AllowRedirect
	RequireRedirect
//	RequireRejoin
	FilamentStart
	FilamentEnd
	Branched
	SideEffect
	Recap // can follow up any record
)

type PrecedenceGroup uint8

const (
	_  PrecedenceGroup = iota
	DropStartGroup
	DropConnectGroup
	DropGeneralGroup

	LineActivateAbleGroup
	LineActivatedGroup
	LineStartGroup
	LineInboundGroup

	InboundFilamentGroup
	OpenOutboundGroup
)

func p(fp FieldPolicy, memberOf PrecedenceGroup, canFollow... PrecedenceGroup) (r RecordPolicy) {
	r.FieldPolicy = fp
	r.MemberOf = memberOf
	for _, m := range canFollow {
		r.CanFollow |= 1<<m
	}
	return
}

var policies = []RecordPolicy{
	rms.TypeRLifelineStartPolymorthID: p(0, LineStartGroup),
	rms.TypeRSidelineStartPolymorthID: p(Branched, LineStartGroup),

	rms.TypeRLineInboundRequestPolymorthID: p(FilamentStart, LineInboundGroup),
	rms.TypeRInboundRequestPolymorthID: p(FilamentStart|Branched, InboundFilamentGroup, LineActivatedGroup),

	rms.TypeROutboundRequestPolymorthID: p(0, OpenOutboundGroup, InboundFilamentGroup, LineInboundGroup),
	rms.TypeROutboundResponsePolymorthID: p(0, InboundFilamentGroup, OpenOutboundGroup),
	rms.TypeRInboundResponsePolymorthID: p(FilamentEnd, 0, InboundFilamentGroup, OpenOutboundGroup),


	rms.TypeRLineActivatePolymorthID: p(SideEffect, LineActivatedGroup, LineActivateAbleGroup, LineStartGroup),
	rms.TypeRLineDeactivatePolymorthID: p(FilamentEnd|SideEffect, 0),

	rms.TypeRLineMemoryInitPolymorthID: p(0, LineActivateAbleGroup, LineStartGroup),
	rms.TypeRLineMemoryPolymorthID: p(SideEffect, 0, LineInboundGroup, LineStartGroup),
	rms.TypeRLineMemoryReusePolymorthID: p(SideEffect, 0, LineInboundGroup), // redirectTo(TypeRLineMemoryPolymorthID)
	rms.TypeRLineMemoryExpectedPolymorthID: p(SideEffect, 0, LineInboundGroup), // blockUntil(TypeRLineMemoryPolymorthID)

	rms.TypeRLineRecapPolymorthID: p(Recap, 0),
}

func GetRecordPolicy(recordType uint64) RecordPolicy {
	if recordType >= uint64(len(policies)) {
		return RecordPolicy{}
	}
	return policies[recordType]
}

func checkRecord(recordType uint64) {
	switch recordType {
	case rms.TypeRLifelineStartPolymorthID:
		// root is empty, prev is empty
	}

>>>>>>> Further work
}
