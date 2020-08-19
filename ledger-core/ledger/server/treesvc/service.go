// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package treesvc

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Service interface {
	// GetTrees provides previous and current jet trees. For genesis-state both trees will be empty.
	// Pulse number is required to enforce synchronicity between the service and the caller.
	// When current pulse number is mismatched, then (found)==false and trees are empty.
	GetTrees(pulse.Number) (prev, cur jet.PrefixTree, found bool)

	// FinishGenesis must be called after genesis procedure is finished with (depth) is a depth of
	// jet tree to be used for normal operations (like multi-split) and (lastGenesisPulse) is the pulse
	// when genesis was yet active.
	FinishGenesis(depth uint8, lastGenesisPulse pulse.Number)

	// IsGenesisFinished returns true after FinishGenesis is called even without a pulse change.
	IsGenesisFinished() bool

	// SplitNext registers a split operation to be applied at next pulse.
	// Only one split or merge operation can be applied to a jet per pulse.
	SplitNext(jet.DropID)

	// MergeNext registers a merge operation to be applied at next pulse.
	// Only one split or merge operation can be applied to a jet per pulse.
	// MergeNext(jet.DropID)
}
