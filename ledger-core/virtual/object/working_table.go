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
	lists map[contract.InterferenceFlag]*WorkingList
}

func NewWorkingTable() WorkingTable {
	var rt WorkingTable
	rt.lists = make(map[contract.InterferenceFlag]*WorkingList)

	rt.lists[contract.CallTolerable] = NewWorkingList()
	rt.lists[contract.CallIntolerable] = NewWorkingList()
	return rt
}

func (rt *WorkingTable) GetList(flag contract.InterferenceFlag) *WorkingList {
	if flag.IsZero() {
		panic(throw.IllegalValue())
	}
	return rt.lists[flag]
}

func (rt *WorkingTable) Len() int {
	size := 0
	for _, list := range rt.lists {
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

type workingRequest struct {
	state  WorkingRequestState
	result *payload.VCallResult
}

type WorkingList struct {
	earliestActivePulse pulse.Number
	countActive         int
	countFinish         int
	requests            map[reference.Global]workingRequest
}

func NewWorkingList() *WorkingList {
	return &WorkingList{
		requests: make(map[reference.Global]workingRequest),
	}
}

func (rl WorkingList) GetState(ref reference.Global) WorkingRequestState {
	return rl.requests[ref].state
}

func (rl WorkingList) GetResult(ref reference.Global) (*payload.VCallResult, bool) {
	if res := rl.requests[ref].result; res != nil {
		return res, true
	}
	return nil, false
}

// Add adds reference.Global and update EarliestPulse if needed
// returns true if added and false if already exists
func (rl *WorkingList) Add(ref reference.Global) bool {
	if _, exist := rl.requests[ref]; exist {
		return false
	}
	rl.requests[ref] = workingRequest{state: RequestStarted}
	return true
}

func (rl *WorkingList) SetActive(ref reference.Global) bool {
	if rl.requests[ref].state != RequestStarted {
		return false
	}

	rl.requests[ref] = workingRequest{state: RequestProcessing}

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
		if rl.requests[ref].state != RequestProcessing {
			continue // skip finished and not started
		}

		refPulseNumber := ref.GetLocal().GetPulseNumber()
		if min == pulse.Unknown || refPulseNumber < min {
			min = refPulseNumber
		}
	}

	rl.earliestActivePulse = min
}

func (rl *WorkingList) Finish(ref reference.Global, result *payload.VCallResult) bool {
	state := rl.GetState(ref)
	if state == RequestUnknown {
		return false
	}

	rl.requests[ref] = workingRequest{state: RequestFinished, result: result}
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
