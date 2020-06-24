// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
<<<<<<< HEAD
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datareader"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type DropSharedData struct {
	id      jet.LegID
	updater buildersvc.JetDropAssistant
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

func (p *DropSharedData) GetDropAssistant() buildersvc.JetDropAssistant {
	p.ensureAccess()
	if p.updater == nil {
		panic(throw.IllegalState())
	}
	return p.updater
}

func (p *DropSharedData) addPrevReport(report datareader.PrevDropReport) {
=======
)

type DropSharedData struct {

}

func (p *DropSharedData) GetReadySync() smachine.SyncLink {
>>>>>>> Ledger SMs
=======
)

type DropSharedData struct {

}

func (p *DropSharedData) GetReadySync() smachine.SyncLink {
>>>>>>> Ledger SMs

}
