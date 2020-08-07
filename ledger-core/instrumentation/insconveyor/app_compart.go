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

type AppCompartmentSetup struct {
	ConveyorConfig     conveyor.PulseConveyorConfig
	ConveyorCycleFn    conveyor.PulseConveyorCycleFunc
	EventFactoryFn     conveyor.PulseEventFactoryFunc
	Components         []managed.Component
	Dependencies       *injector.DynamicContainer
}

func (p *AppCompartmentSetup) AddComponent(c managed.Component) {
	p.Components = append(p.Components, c)
}

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
	if p.imposeFn != nil {
		params := ImposedParams{
			CompartmentSetup:  appCfg,
		}
		p.imposeFn(&params)
		p.imposeFn = nil

		appCfg = params.CompartmentSetup
		interceptFn = params.ComponentInterceptFn
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

	p.conveyorWorker = NewConveyorWorker(appCfg.ConveyorCycleFn)
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

