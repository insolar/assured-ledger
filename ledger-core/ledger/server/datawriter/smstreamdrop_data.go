// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
<<<<<<< HEAD
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
=======
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)


type StreamSharedData struct {
	ready smsync.BoolConditionalLink

	state atomickit.Uint32
	pr    pulse.Range
<<<<<<< HEAD
<<<<<<< HEAD
	jetAssist buildersvc.StreamDropAssistant
=======
>>>>>>> Ledger SMs
=======
>>>>>>> Ledger SMs
	// jetTree
	// population
}

func (p *StreamSharedData) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

<<<<<<< HEAD
<<<<<<< HEAD
func (p *StreamSharedData) enableAccess() smachine.SyncAdjustment {
	p.state.Store(1)
	return p.ready.NewValue(true)
=======
func (p *StreamSharedData) enableAccess() {
	p.state.Store(1)
>>>>>>> Ledger SMs
=======
func (p *StreamSharedData) enableAccess() {
	p.state.Store(1)
>>>>>>> Ledger SMs
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

<<<<<<< HEAD
<<<<<<< HEAD
func (p *StreamSharedData) GetJetDrop(ref reference.Holder) jet.DropID {
	p.ensureAccess()
	return p.jetAssist.CalculateJetDrop(ref)
=======
func (p *StreamSharedData) GetJetDrop(ref reference.Global) JetDropID {
	p.ensureAccess()
>>>>>>> Ledger SMs
=======
func (p *StreamSharedData) GetJetDrop(ref reference.Global) JetDropID {
	p.ensureAccess()
>>>>>>> Ledger SMs
}
