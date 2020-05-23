package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type PendingTable struct {
	Ordered   PendingList
	Unordered PendingList
}

type PendingList struct {
	oldestPulse pulse.Number
	requests    map[reference.Global]isActive
}

type isActive bool

func NewPendingTable() PendingTable {
	return PendingTable{
		Ordered:   NewPendingList(),
		Unordered: NewPendingList(),
	}
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

	pt.requests[ref] = false

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
