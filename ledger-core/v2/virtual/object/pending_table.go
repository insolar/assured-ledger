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

func (pl *PendingList) Exist(ref reference.Global) bool {
	_, exist := pl.requests[ref]
	return exist
}

// Add adds reference.Global and update OldestPulse if needed
// returns true if added and false if already exists
func (pl *PendingList) Add(ref reference.Global) bool {
	if pl.Exist(ref) {
		return false
	}

	pl.requests[ref] = true

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	if pl.oldestPulse == 0 || requestPulseNumber < pl.oldestPulse {
		pl.oldestPulse = requestPulseNumber
	}

	return true
}

func (pl *PendingList) Finish(ref reference.Global) bool {
	if !pl.Exist(ref) {
		return false
	}

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	pl.requests[ref] = false

	if requestPulseNumber != pl.oldestPulse {
		return true
	}

	min := pulse.Unknown
	for ref := range pl.requests {
		// skip finished
		if !pl.requests[ref] {
			continue
		}
		refPulseNumber := ref.GetLocal().GetPulseNumber()
		if min == pulse.Unknown || refPulseNumber < min {
			min = refPulseNumber
		}
	}

	pl.oldestPulse = min

	return true
}

func (pl *PendingList) Count() uint8 {
	return uint8(len(pl.requests))
}

func (pl *PendingList) CountFinish() uint8 {
	var count uint8
	for _, requestIsActive := range pl.requests {
		if !requestIsActive {
			count++
		}
	}
	return count
}

func (pl *PendingList) CountActive() uint8 {
	var count uint8
	for _, requestIsActive := range pl.requests {
		if requestIsActive {
			count++
		}
	}
	return count
}

func (pl *PendingList) OldestPulse() pulse.Number {
	return pl.oldestPulse
}
