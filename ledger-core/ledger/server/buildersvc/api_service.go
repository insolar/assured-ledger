// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Service interface {
	CreatePlash(pr pulse.Range, treePrev, treeCur jet.Tree, online census.OnlinePopulation) (PlashAssistant, []jet.ExactID)
	CreateGenesis(pulse.Range, census.OnlinePopulation) (PlashAssistant, jet.ExactID)
	AppendToDrop(jet.DropID, AppendFuture, lineage.UpdateBundle)
	AppendToDropSummary(jet.DropID, lineage.LineSummary)
	FinalizeDropSummary(jet.DropID) catalog.DropReport
}

type PlashAssistant interface {
	PreparePulseChange(outFn conveyor.PreparePulseCallbackFunc)
	CancelPulseChange()
	CommitPulseChange()

	CalculateJetDrop(reference.Holder) jet.DropID
	IsGenesis() bool

	// GetNextPlashReadySync returns a sync object that will be opened after next Plash is started.
	GetNextPlashReadySync() smachine.SyncLink
	// GetNextPlash must NOT be accessed before GetNextPlashReadySync() is triggered.
	GetNextPlash() PlashAssistant

	// TODO removed GetNextPlash, but enable CalculateNextDrop(Executor)(prevDrop jet.DropID) ([]nodeId)
}

type AppendFuture interface {
	TrySetFutureResult(allocations []ledger.DirectoryIndex, err error) bool
}