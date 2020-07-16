// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type DropSharedData struct {
	id      jet.DropID
	ready   smsync.BoolConditionalLink
	prevReport smachine.BargeInWithParam

	state atomickit.Uint32
}

func (p *DropSharedData) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

func (p *DropSharedData) enableAccess() smachine.SyncAdjustment {
	p.state.Store(1)
	return p.ready.NewValue(true)
}

func (p *DropSharedData) ensureAccess() {
	if p.state.Load() == 0 {
		panic(throw.IllegalState())
	}
}

func (p *DropSharedData) SetPrevDropReport(report datareader.PrevDropReport) {
	p.prevReport.CallWithParam(report)
}

func (p *DropSharedData) addPrevReport(datareader.PrevDropReport) {
	p.ensureAccess()
	// TODO
}
