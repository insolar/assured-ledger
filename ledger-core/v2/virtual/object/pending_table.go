package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type pendingTable struct {
	Ordered   pendingList
	Unordered pendingList
}

type pendingList struct {
	oldestPulse pulse.Number
	requests    map[reference.Global]*struct {
		active bool
	}
}

// nolint // it will be used in PLAT-311
func newPendingTable() pendingTable {
	return pendingTable{
		Ordered:   newPendingList(),
		Unordered: newPendingList(),
	}
}

func newPendingList() pendingList {
	return pendingList{
		requests: make(map[reference.Global]*struct{ active bool }),
	}
}

// Add adds reference.Global and update OldestPulse if needed
// returns true if added and false if already exists
func (pt *pendingList) Add(ref reference.Global) bool {
	if _, exist := pt.requests[ref]; exist {
		return false
	}

	pt.requests[ref] = &struct{ active bool }{true}

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	if pt.oldestPulse == 0 || requestPulseNumber < pt.oldestPulse {
		pt.oldestPulse = requestPulseNumber
	}

	return true
}

func (pt *pendingList) Finish(ref reference.Global) bool {
	if _, exist := pt.requests[ref]; !exist {
		return false
	}
	pt.requests[ref].active = false

	return true
}

func (pt *pendingList) Count() int {
	return len(pt.requests)
}

func (pt *pendingList) CountFinish() int {
	var count int
	for _, request := range pt.requests {
		if !request.active {
			count++
		}
	}
	return count
}

func (pt *pendingList) OldestPulse() pulse.Number {
	return pt.oldestPulse
}
