// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package callregistry

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type WorkingTable struct {
	requests [isolation.InterferenceFlagCount - 1]*WorkingList
	results  map[reference.Global]CallSummary
}

func NewWorkingTable() WorkingTable {
	var rt WorkingTable

	for i := len(rt.requests) - 1; i >= 0; i-- {
		rt.requests[i] = newWorkingList()
	}

	rt.results = make(map[reference.Global]CallSummary)

	return rt
}

func (wt WorkingTable) GetList(flag isolation.InterferenceFlag) *WorkingList {
	if flag > 0 && int(flag) <= len(wt.requests) {
		return wt.requests[flag-1]
	}
	panic(throw.IllegalValue())
}

func (wt WorkingTable) GetResults() map[reference.Global]CallSummary {
	return wt.results
}

// Add adds reference.Global
// returns true if added and false if already exists
func (wt WorkingTable) Add(flag isolation.InterferenceFlag, ref reference.Global) bool {
	return wt.GetList(flag).add(ref)
}

func (wt WorkingTable) SetActive(flag isolation.InterferenceFlag, ref reference.Global) bool {
	if ok := wt.GetList(flag).setActive(ref); ok {
		wt.results[ref] = CallSummary{}

		return true
	}

	return false
}

func (wt WorkingTable) Finish(
	flag isolation.InterferenceFlag,
	ref reference.Global,
	result *payload.VCallResult,
) bool {
	if ok := wt.GetList(flag).finish(ref); ok {
		_, ok := wt.results[ref]
		if !ok {
			panic(throw.IllegalState())
		}
		wt.results[ref] = CallSummary{Result: result}

		return true
	}

	return false
}

func (wt *WorkingTable) Len() int {
	size := 0
	for _, list := range wt.requests[1:] {
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
	activity            map[payload.PulseNumber]uint
}

func newWorkingList() *WorkingList {
	return &WorkingList{
		requests: make(map[reference.Global]WorkingRequestState),
		activity: make(map[payload.PulseNumber]uint),
	}
}

type CallSummary struct {
	Result *payload.VCallResult
}

func (rl *WorkingList) GetState(ref reference.Global) WorkingRequestState {
	return rl.requests[ref]
}

func (rl *WorkingList) add(ref reference.Global) bool {
	if _, exist := rl.requests[ref]; exist {
		return false
	}
	rl.requests[ref] = RequestStarted
	return true
}

func (rl *WorkingList) setActive(ref reference.Global) bool {
	if rl.requests[ref] != RequestStarted {
		return false
	}

	rl.requests[ref] = RequestProcessing

	rl.listActivity(ref.GetLocal().GetPulseNumber())

	return true
}

func (rl *WorkingList) finish(ref reference.Global) bool {
	state := rl.GetState(ref)
	if state == RequestUnknown || state == RequestFinished {
		return false
	}

	rl.requests[ref] = RequestFinished
	rl.countFinish++

	rl.delistActivity(ref.GetLocal().GetPulseNumber())

	return true
}

func (rl *WorkingList) listActivity(requestPN pulse.Number) {
	rl.countActive++
	rl.activity[requestPN]++
	if rl.earliestActivePulse == pulse.Unknown || requestPN < rl.earliestActivePulse {
		rl.earliestActivePulse = requestPN
	}
}

func (rl *WorkingList) delistActivity(requestPN pulse.Number) {
	rl.countActive--

	count := rl.activity[requestPN]
	if count > 1 {
		rl.activity[requestPN] = count - 1
	} else {
		delete(rl.activity, requestPN)
		if requestPN == rl.earliestActivePulse {
			rl.earliestActivePulse = rl.minActivePulse()
		}
	}
}

func (rl *WorkingList) minActivePulse() pulse.Number {
	min := pulse.Unknown
	for p := range rl.activity {
		if min == pulse.Unknown || p < min {
			min = p
		}
	}
	return min
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
