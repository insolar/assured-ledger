// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type DropSharedData struct {
	info DropInfo

	ready             smsync.BoolConditionalLink
	finalize          smachine.SyncLink
	prevReportBargein smachine.BargeInWithParam

	state atomickit.Uint32
}

type DropInfo struct {
	ID jet.DropID
	PrevID jet.DropID
	LastOp JetTreeOp
	// LegID jet.LegID // TODO LegID
	AssistData *PlashSharedData
}


func (p *DropSharedData) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

func (p *DropSharedData) enableAccess() smachine.SyncAdjustment {
	p.state.Store(1)
	return p.ready.NewValue(true)
}

//nolint
func (p *DropSharedData) ensureAccess() {
	if p.state.Load() == 0 {
		panic(throw.IllegalState())
	}
}

func (p *DropSharedData) SetPrevDropReport(report catalog.DropReport) {
	p.prevReportBargein.CallWithParam(report)
}

func (p *DropSharedData) setPrevReport(catalog.DropReport) {
	// TODO
}

func (p *DropSharedData) GetFinalizeSync() smachine.SyncLink {
	return p.finalize
}
