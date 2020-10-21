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

type BasicPlashConfig struct {
	PulseRange pulse.Range
	Population census.OnlinePopulation
	CallbackFn func(error)
}

type Service interface {
	CreatePlash(cfg BasicPlashConfig, treePrev, treeCur jet.Tree) (PlashAssistant, conveyor.PulseChanger, []jet.ExactID)
	CreateGenesis(BasicPlashConfig) (PlashAssistant, conveyor.PulseChanger, jet.ExactID)
	AppendToDrop(jet.DropID, AppendFuture, lineage.UpdateBundle)
	AppendToDropSummary(jet.DropID, lineage.LineSummary)
	FinalizeDropSummary(jet.DropID) catalog.DropReport
}

type PlashAssistant interface {
	CalculateJetDrop(reference.Holder) jet.DropID
	IsGenesis() bool

	// CalculateDropNodes(jet.DropID) (executorID, validatorID)

	// GetNextReadySync returns a sync object that will be opened after next Plash is started.
	GetNextReadySync() smachine.SyncLink
	// CalculateNextDrops must NOT be accessed before GetNextReadySync() is triggered.
	CalculateNextDrops(jet.DropID) []jet.DropID
}

type AppendFuture interface {
	TrySetFutureResult(allocations []ledger.DirectoryIndex, err error) bool
}
