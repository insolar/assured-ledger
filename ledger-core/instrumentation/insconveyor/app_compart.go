// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ComponentSetupFunc = func(context.Context, injector.DependencyInjector, managed.RegisterComponentFunc)

func NewAppCompartment(name string, appDeps injector.DependencyRegistry, setupFn ConfigReviserFunc) *AppCompartment {
	switch {
	case name == "":
		panic(throw.IllegalValue())
	case appDeps == nil:
		panic(throw.IllegalValue())
	}

	return &AppCompartment{	name: name,	appDeps: appDeps, setupFn: setupFn }
}

// AppCompartmentSetup is a setup of an app compartment.
type AppCompartmentSetup struct {
	// ConveyorConfig provides settings for conveyor.
	ConveyorConfig     conveyor.PulseConveyorConfig
	// EventFactoryFn provides event factory for conveyor.
	EventFactoryFn     conveyor.PulseEventFactoryFunc
	// Components will be added with conveyor.PulseConveyor.AddManagedComponent().
	Components         []managed.Component
	// Dependencies will be added as dependencies to conveyor.PulseConveyor.
	Dependencies       *injector.DynamicContainer
}

// AddComponent is a convenience method to add to the Components field.
func (p *AppCompartmentSetup) AddComponent(c managed.Component) {
	p.Components = append(p.Components, c)
}

// ConfigReviserFunc enables AppCompartmentSetup to be revised before creation of an app compartment.
type ConfigReviserFunc = func (context.Context, injector.DependencyInjector, AppCompartmentSetup) AppCompartmentSetup

var _ insapp.AppComponent = &AppCompartment{}
type AppCompartment struct {
	// set by construction
	name     string
	appDeps  injector.DependencyRegistry
	setupFn  ConfigReviserFunc
	imposeFn ImposerFunc

	// dependencies, resolved by Init

	ctx            context.Context
	flowDispatcher beat.Dispatcher
	conveyor       *conveyor.PulseConveyor
	conveyorWorker ConveyorWorker
}

// SetImposer sets a handler that can alternate Init() behavior. For tests ONLY.
func (p *AppCompartment) SetImposer(fn ImposerFunc) {
	if p.conveyor != nil {
		panic(throw.IllegalState())
	}
	p.imposeFn = fn
}

func (p *AppCompartment) Init(ctx context.Context) error {
	ctx, _ = inslogger.WithField(ctx, "compartment", p.name)
	p.ctx = ctx

	appCfg := AppCompartmentSetup{
		ConveyorConfig: DefaultConfig(),
		Dependencies: injector.NewDynamicContainer(nil),
	}

	inject := injector.NewDependencyInjector(struct {}{}, p.appDeps, nil)
	p.appDeps = nil

	overrides := ConfigOverrides{}
	_ = inject.Inject(&overrides.MachineLogger)
	ApplyConfigOverrides(&appCfg.ConveyorConfig, overrides)

	if p.setupFn != nil {
		appCfg = p.setupFn(ctx, inject, appCfg)
	}
	p.setupFn = nil

	var interceptFn ComponentInterceptFunc
	var cycleFn conveyor.PulseConveyorCycleFunc
	if p.imposeFn != nil {
		params := ImposedParams{
			InitContext: ctx,
			CompartmentSetup:  appCfg,
		}
		p.imposeFn(&params)
		p.imposeFn = nil

		appCfg = params.CompartmentSetup
		interceptFn = params.ComponentInterceptFn
		cycleFn = params.ConveyorCycleFn
	}

	p.conveyor = conveyor.NewPulseConveyor(ctx,
		appCfg.ConveyorConfig,
		appCfg.EventFactoryFn,
		appCfg.Dependencies.CopyAsStatic().AsRegistry(), // can handle nil
	)

	var addFn managed.RegisterComponentFunc
	if interceptFn != nil {
		addFn = func(c managed.Component) {
			if c = interceptFn(c); c != nil {
				p.conveyor.AddManagedComponent(c)
			}
		}
	} else {
		addFn = p.conveyor.AddManagedComponent
	}

	for _, c := range appCfg.Components {
		if c != nil {
			addFn(c)
		}
	}

	p.conveyorWorker = NewConveyorWorker(cycleFn)
	p.flowDispatcher = NewConveyorDispatcher(ctx, p.conveyor)

	return nil
}

func (p *AppCompartment) Start(context.Context) error {
	p.conveyorWorker.AttachTo(p.conveyor)
	return nil
}

func (p *AppCompartment) Stop(context.Context) error {
	p.conveyorWorker.Stop()
	return nil
}

func (p *AppCompartment) AppDependencies() injector.DependencyRegistry {
	return p.appDeps
}

func (p *AppCompartment) Conveyor() *conveyor.PulseConveyor {
	return p.conveyor
}

func (p *AppCompartment) GetMessageHandler() message.NoPublishHandlerFunc {
	return p.flowDispatcher.Process
}

func (p *AppCompartment) GetBeatDispatcher() beat.Dispatcher {
	return p.flowDispatcher
}

