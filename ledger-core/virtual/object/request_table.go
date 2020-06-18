// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RequestTable struct {
	lists map[contract.InterferenceFlag]*RequestList
}

func NewRequestTable() RequestTable {
	var rt RequestTable
	rt.lists = make(map[contract.InterferenceFlag]*RequestList)

	rt.lists[contract.CallTolerable] = NewRequestList()
	rt.lists[contract.CallIntolerable] = NewRequestList()
	return rt
}

func (rt *RequestTable) GetList(flag contract.InterferenceFlag) *RequestList {
	if flag.IsZero() {
		panic(throw.IllegalValue())
	}
	return rt.lists[flag]
}

func (rt *RequestTable) Len() int {
	size := 0
	for _, list := range rt.lists {
		size += list.Count()
	}
	return size
}

type isActive bool

type RequestList struct {
	earliestPulse pulse.Number
	countActive   int
	countFinish   int
	requests      map[reference.Global]isActive
}

func NewRequestList() *RequestList {
	return &RequestList{
		requests: make(map[reference.Global]isActive),
	}
}

func (rl RequestList) Exist(ref reference.Global) bool {
	_, exist := rl.requests[ref]
	return exist
}

// Add adds reference.Global and update EarliestPulse if needed
// returns true if added and false if already exists
func (rl *RequestList) Add(ref reference.Global) bool {
	if _, exist := rl.requests[ref]; exist {
		return false
	}

	rl.requests[ref] = true
	rl.countActive++

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	if rl.earliestPulse == 0 || requestPulseNumber < rl.earliestPulse {
		rl.earliestPulse = requestPulseNumber
	}

	return true
}

func (rl *RequestList) calculateEarliestPulse() {
	min := pulse.Unknown

	for ref := range rl.requests {
		if !rl.requests[ref] {
			continue // skip finished
		}

		refPulseNumber := ref.GetLocal().GetPulseNumber()
		if min == pulse.Unknown || refPulseNumber < min {
			min = refPulseNumber
		}
	}

	rl.earliestPulse = min
}

func (rl *RequestList) Finish(ref reference.Global) bool {
	if !rl.Exist(ref) {
		return false
	}

	rl.requests[ref] = false
	rl.countActive--
	rl.countFinish++

	if ref.GetLocal().GetPulseNumber() == rl.earliestPulse {
		rl.calculateEarliestPulse()
	}

	return true
}

func (rl *RequestList) Count() int {
	return len(rl.requests)
}

func (rl *RequestList) CountFinish() int {
	return rl.countFinish
}

func (rl *RequestList) CountActive() int {
	return rl.countActive
}

func (rl *RequestList) EarliestPulse() pulse.Number {
	return rl.earliestPulse
}
