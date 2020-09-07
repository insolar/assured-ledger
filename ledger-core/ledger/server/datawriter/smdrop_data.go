// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datafinder"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type DropSharedData struct {
	id                jet.DropID
	ready             smsync.BoolConditionalLink
	prevReportBargein smachine.BargeInWithParam

	state atomickit.Uint32
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

func (p *DropSharedData) SetPrevDropReport(report datafinder.PrevDropReport) {
	p.prevReportBargein.CallWithParam(report)
}

func (p *DropSharedData) setPrevReport(datafinder.PrevDropReport) {
	// TODO
}
