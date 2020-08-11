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
	GetTrees() (prev, cur jet.PrefixTree)

	FinishGenesis(depth uint8, afterPulse pulse.Number)

	SplitNext(jet.DropID)
	// MergeNext(jet.DropID)
}
