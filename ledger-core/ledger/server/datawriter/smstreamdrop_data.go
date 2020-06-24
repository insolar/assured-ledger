// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)


type StreamSharedData struct {
	ready smsync.BoolConditionalLink

	state atomickit.Uint32
	pr    pulse.Range
	// jetTree
	// population
}

func (p *StreamSharedData) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

func (p *StreamSharedData) enableAccess() {
	p.state.Store(1)
}

func (p *StreamSharedData) ensureAccess() {
	if p.state.Load() == 0 {
		panic(throw.IllegalState())
	}
}

func (p *StreamSharedData) GetPulseRange() pulse.Range {
	p.ensureAccess()
	if p.pr == nil {
		panic(throw.IllegalState())
	}
	return p.pr
}

func (p *StreamSharedData) GetJetDrop(ref reference.Global) JetDropID {
	p.ensureAccess()
}
