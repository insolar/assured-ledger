// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package managed

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

// Component is a component managed on conveyor's level.
// It has access to conveyor's internal SlotMachine, but not to individual pulse slots.
type Component interface {
	// Init is called when a component is added to a conveyor.
	// NB! When calls to Start() or Stop() may happen concurrently with Init() when the component is added in a separate goroutine.
	Init(Holder)
	// Start is called on component when Conveyor's worker is started, or when a component is added to a started conveyor
	// NB! When conveyor is already started, call to Stop() may happen before completion of Start()
	Start(Holder)
	Stop(Holder)
}

// ComponentWithPulse enables a component to be notified during pulse migration.
type ComponentWithPulse interface {
	Component
	// PulseMigration is called during pulse migration procedure of the conveyor.
	// It is invoked after all internal operations of the conveyor, but before migration of pulse slots.
	// MUST NOT perform heavy operations as the whole conveyor is blocked.
	PulseMigration(Holder, pulse.Range)
}

type RegisterComponentFunc = func(Component)

type Holder interface {
	GetDataManager() DataManager
	AddManagedComponent(Component)
	AddInputExt(pn pulse.Number, event interface{}, createDefaults smachine.CreateDefaultValues) error
	WakeUpWorker()

	injector.DependencyContainer
	AddDependency(v interface{})
	AddInterfaceDependency(v interface{})

	GetPublishedGlobalAliasAndBargeIn(key interface{}) (smachine.SlotLink, smachine.BargeInHolder)
}

type DataManager interface {
	GetPresentPulse() (present pulse.Number, nearestFuture pulse.Number)
	GetPrevBeatData() (pulse.Number, BeatData)
	GetBeatData(pn pulse.Number) BeatData
	HasPulseData(pn pulse.Number) bool
	TouchPulseData(pn pulse.Number) bool
}

type BeatData struct {
	Range  pulse.Range
	Online census.OnlinePopulation
}


