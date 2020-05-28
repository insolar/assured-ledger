// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type PendingTable struct {
	lists map[contract.InterferenceFlag]PendingList
}

func NewPendingTable() PendingTable {
	var pt PendingTable
	pt.lists = make(map[contract.InterferenceFlag]PendingList)

	pt.lists[contract.CallTolerable] = NewPendingList()
	pt.lists[contract.CallIntolerable] = NewPendingList()
	return pt
}

func (pt *PendingTable) GetList(flag contract.InterferenceFlag) PendingList {
	if flag != contract.CallTolerable && flag != contract.CallIntolerable {
		panic(throw.IllegalValue())
	}
	return pt.lists[flag]
}

type isActive bool

type PendingList struct {
	oldestPulse pulse.Number
	requests    map[reference.Global]isActive
}

func NewPendingList() PendingList {
	return PendingList{
		requests: make(map[reference.Global]isActive),
	}
}

// Add adds reference.Global and update OldestPulse if needed
// returns true if added and false if already exists
func (pt *PendingList) Add(ref reference.Global) bool {
	if _, exist := pt.requests[ref]; exist {
		return false
	}

	pt.requests[ref] = true

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	if pt.oldestPulse == 0 || requestPulseNumber < pt.oldestPulse {
		pt.oldestPulse = requestPulseNumber
	}

	return true
}

func (pt *PendingList) Finish(ref reference.Global) bool {
	if _, exist := pt.requests[ref]; !exist {
		return false
	}

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	pt.requests[ref] = false

	if requestPulseNumber != pt.oldestPulse {
		return true
	}

	min := pulse.Unknown
	for ref := range pt.requests {
		// skip finished
		if !pt.requests[ref] {
			continue
		}
		refPulseNumber := ref.GetLocal().GetPulseNumber()
		if min == pulse.Unknown || refPulseNumber < min {
			min = refPulseNumber
		}
	}

	pt.oldestPulse = min

	return true
}

func (pt *PendingList) Count() int {
	return len(pt.requests)
}

func (pt *PendingList) CountFinish() int {
	var count int
	for _, requestIsActive := range pt.requests {
		if !requestIsActive {
			count++
		}
	}
	return count
}

func (pt *PendingList) OldestPulse() pulse.Number {
	return pt.oldestPulse
}
