package object

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type pendingTable struct {
	oldestPulse pulse.Number
	requests    map[reference.Global]struct{}
}

func newPendingTable() pendingTable {
	return pendingTable{requests: make(map[reference.Global]struct{})}
}

// Add adds reference.Global and update OldestPulse if needed
// returns true if added and false if already exists
func (pt *pendingTable) Add(ref reference.Global) bool {
	if _, exist := pt.requests[ref]; exist {
		return false
	}

	pt.requests[ref] = struct{}{}

	requestPulseNumber := ref.GetLocal().GetPulseNumber()
	if pt.oldestPulse == 0 || requestPulseNumber < pt.oldestPulse {
		pt.oldestPulse = requestPulseNumber
	}

	return true
}

func (pt *pendingTable) Count() int {
	return len(pt.requests)
}
