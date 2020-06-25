// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type LineKey reference.Global

type LineSharedData struct {
	lineRef reference.Global
	limiter smsync.SemaphoreLink
	ready   bool
	valid   bool

	jetDropID   JetDropID
	dropUpdater buildersvc.JetDropAssistant
	recapRef    reference.Global
}

func (p *LineSharedData) LineRef() reference.Global {
	return p.lineRef
}

func (p *LineSharedData) GetLimiter() smachine.SyncLink {
	return p.limiter.SyncLink()
}

func (p *LineSharedData) TryApplyRecordSet(set inspectsvc.InspectedRecordSet) (*buildersvc.Future, interface{}) {
	p.ensureAccess()
}

func (p *LineSharedData) RequestMissings(missings interface{}, wakeup smachine.BargeInHolder) {
	p.ensureAccess()
}

func (p *LineSharedData) ensureAccess() {
	if !p.ready {
		panic(throw.IllegalState())
	}
}

func (p *LineSharedData) IsValid() bool {
	p.ensureAccess()
	return p.valid
}

func (p *LineSharedData) enableAccess() smachine.SyncAdjustment {
	p.ready = true
	return p.limiter.NewValue(1)
}

func (p *LineSharedData) disableAccess() {
	p.ready = false
}

func (p *LineSharedData) addRecap(rec *rms.RLineRecap, ref reference.Global) {

}

func (p *LineSharedData) getMissingFilaments() []reference.Global {

}
