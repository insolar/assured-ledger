// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package callregistry

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PendingTable struct {
	lists [isolation.InterferenceFlagCount - 1]*PendingList
}

func NewRequestTable() PendingTable {
	var rt PendingTable

	for i := len(rt.lists) - 1; i >= 0; i-- {
		rt.lists[i] = newRequestList()
	}

	return rt
}

func (rt PendingTable) GetList(flag isolation.InterferenceFlag) *PendingList {
	if flag > 0 && int(flag) <= len(rt.lists) {
		return rt.lists[flag-1]
	}
	panic(throw.IllegalValue())
}

func (rt PendingTable) Len() int {
	size := 0
	for _, list := range rt.lists {
		size += list.Count()
	}
	return size
}

type isActive bool

type PendingList struct {
	earliestActivePulse pulse.Number
	countActive         int
	countFinish         int
	requests            map[reference.Global]isActive
	activity            map[payload.PulseNumber]uint
}

func newRequestList() *PendingList {
	return &PendingList{
		requests: make(map[reference.Global]isActive),
		activity: make(map[payload.PulseNumber]uint),
	}
}

func (rl *PendingList) Exist(ref reference.Global) bool {
	_, exist := rl.requests[ref]
	return exist
}

// returns isActive and Exist info
func (rl PendingList) GetState(ref reference.Global) (bool, bool) {
	if !rl.Exist(ref) {
		return false, false
	}
	return bool(rl.requests[ref]), true
}

// Add adds reference.Global and update EarliestPulse if needed
// returns true if added and false if already exists
func (rl *PendingList) Add(ref reference.Global) bool {
	if _, exist := rl.requests[ref]; exist {
		return false
	}

	rl.requests[ref] = true

	rl.listActivity(ref.GetLocal().GetPulseNumber())

	return true
}

func (rl *PendingList) listActivity(requestPN pulse.Number) {
	rl.countActive++
	rl.activity[requestPN]++
	if rl.earliestActivePulse == pulse.Unknown || requestPN < rl.earliestActivePulse {
		rl.earliestActivePulse = requestPN
	}
}

func (rl *PendingList) delistActivity(requestPN pulse.Number) {
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

func (rl *PendingList) minActivePulse() pulse.Number {
	min := pulse.Unknown
	for p := range rl.activity {
		if min == pulse.Unknown || p < min {
			min = p
		}
	}
	return min
}

func (rl *PendingList) Finish(ref reference.Global) bool {
	if !rl.Exist(ref) {
		return false
	}

	rl.requests[ref] = false
	rl.countFinish++

	rl.delistActivity(ref.GetLocal().GetPulseNumber())

	return true
}

func (rl *PendingList) Count() int {
	return len(rl.requests)
}

func (rl *PendingList) CountFinish() int {
	return rl.countFinish
}

func (rl *PendingList) CountActive() int {
	return rl.countActive
}

func (rl *PendingList) EarliestPulse() pulse.Number {
	return rl.earliestActivePulse
}
