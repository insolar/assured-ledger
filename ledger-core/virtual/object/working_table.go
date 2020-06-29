// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type WorkingTable struct {
	requests []*WorkingList
	results  map[reference.Global]Summary
}

func NewWorkingTable() WorkingTable {
	var rt WorkingTable

	lists := make([]*WorkingList, contract.InterferenceFlagCount)
	for i := 1; i < contract.InterferenceFlagCount; i++ {
		lists[i] = NewWorkingList()
	}

	rt.requests = lists

	return rt
}

func (wt *WorkingTable) GetList(flag contract.InterferenceFlag) *WorkingList {
	if flag.IsZero() {
		panic(throw.IllegalValue())
	}
	return wt.requests[flag]
}

func (wt *WorkingTable) GetResults() map[reference.Global]Summary {
	return wt.results
}

func (wt *WorkingTable) Add(flag contract.InterferenceFlag, ref reference.Global) bool {
	return wt.GetList(flag).Add(ref)
}

func (wt *WorkingTable) SetActive(flag contract.InterferenceFlag, ref reference.Global) bool {
	if ok := wt.GetList(flag).SetActive(ref); ok {
		wt.results[ref] = Summary{}

		return true
	}

	return false
}

func (wt *WorkingTable) Finish(
	flag contract.InterferenceFlag,
	ref reference.Global,
	result *payload.VCallResult,
) bool {
	if ok := wt.GetList(flag).Finish(ref); ok {
		summary, ok := wt.results[ref]
		if !ok {
			panic(throw.IllegalState())
		}
		summary.result = result

		return true
	}

	return false
}

func (wt *WorkingTable) Len() int {
	size := 0
	for _, list := range wt.requests {
		size += list.Count()
	}
	return size
}

type WorkingRequestState int

const (
	RequestUnknown WorkingRequestState = iota
	RequestStarted
	RequestProcessing
	RequestFinished
)

type WorkingList struct {
	earliestActivePulse pulse.Number
	countActive         int
	countFinish         int
	requests            map[reference.Global]WorkingRequestState
}

func NewWorkingList() *WorkingList {
	return &WorkingList{
		requests: make(map[reference.Global]WorkingRequestState),
	}
}

type Summary struct {
	result *payload.VCallResult
}

func (rl *WorkingList) GetState(ref reference.Global) WorkingRequestState {
	return rl.requests[ref]
}

// Add adds reference.Global and update EarliestPulse if needed
// returns true if added and false if already exists
func (rl *WorkingList) Add(ref reference.Global) bool {
	if _, exist := rl.requests[ref]; exist {
		return false
	}
	rl.requests[ref] = RequestStarted
	return true
}

func (rl *WorkingList) SetActive(ref reference.Global) bool {
	if rl.requests[ref] != RequestStarted {
		return false
	}

	rl.requests[ref] = RequestProcessing

	rl.countActive++

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	if rl.earliestActivePulse == pulse.Unknown || requestPulseNumber < rl.earliestActivePulse {
		rl.earliestActivePulse = requestPulseNumber
	}

	return true
}

func (rl *WorkingList) calculateEarliestActivePulse() {
	min := pulse.Unknown

	for ref := range rl.requests {
		if rl.requests[ref] != RequestProcessing {
			continue // skip finished and not started
		}

		refPulseNumber := ref.GetLocal().GetPulseNumber()
		if min == pulse.Unknown || refPulseNumber < min {
			min = refPulseNumber
		}
	}

	rl.earliestActivePulse = min
}

func (rl *WorkingList) Finish(ref reference.Global) bool {
	state := rl.GetState(ref)
	if state == RequestUnknown {
		return false
	}

	rl.requests[ref] = RequestFinished
	rl.countActive--
	rl.countFinish++

	if ref.GetLocal().GetPulseNumber() == rl.earliestActivePulse {
		rl.calculateEarliestActivePulse()
	}

	return true
}

func (rl *WorkingList) Count() int {
	return len(rl.requests)
}

func (rl *WorkingList) CountFinish() int {
	return rl.countFinish
}

func (rl *WorkingList) CountActive() int {
	return rl.countActive
}

func (rl *WorkingList) EarliestPulse() pulse.Number {
	return rl.earliestActivePulse
}
