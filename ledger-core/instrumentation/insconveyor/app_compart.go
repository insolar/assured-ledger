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
type AppCompartmentSetupFunc = func(context.Context, injector.DependencyInjector) AppCompartmentSetup

func NewAppCompartment(name string, appDeps, convDeps injector.DependencyRegistry,
	componentsFn ComponentSetupFunc, setupFn AppCompartmentSetupFunc,
) *AppCompartment {
	switch {
	case name == "":
		panic(throw.IllegalValue())
	case appDeps == nil:
		panic(throw.IllegalValue())
	}

	return &AppCompartment{	name: name,	appDeps: appDeps, convDeps: convDeps,
		componentsFn: componentsFn, setupFn: setupFn,
	}
}

type AppCompartmentSetup struct {
	ConveyorConfig            conveyor.PulseConveyorConfig
	HasCompleteConveyorConfig bool
	EventFactory              EventFactory
	ConveyorCycleFn           conveyor.PulseConveyorCycleFunc
	ComponentSetupFn          ComponentSetupFunc
}

var _ insapp.AppComponent = &AppCompartment{}
type AppCompartment struct {
	// set by construction
	name         string
	appDeps      injector.DependencyRegistry
	convDeps     injector.DependencyRegistry
	setupFn      AppCompartmentSetupFunc
	componentsFn ComponentSetupFunc

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

	inject := injector.NewDependencyInjector(struct {}{}, p.appDeps, nil)

	var appCfg AppCompartmentSetup
	if p.setupFn != nil {
		appCfg = p.setupFn(ctx, inject)
	}

	if !appCfg.HasCompleteConveyorConfig {
		overrides := ConfigOverrides{}
		_ = inject.Inject(&overrides.MachineLogger)

		pds, psm := appCfg.ConveyorConfig.PulseDataService, appCfg.ConveyorConfig.PulseSlotMigration
		appCfg.ConveyorConfig = DefaultConfigWithOverrides(overrides)

		if pds != nil {
			appCfg.ConveyorConfig.PulseDataService = pds
		}
		if psm != nil {
			appCfg.ConveyorConfig.PulseSlotMigration = psm
		}
	}

	var addFn managed.RegisterComponentFunc

	if p.imposeFn != nil {
		// p.conveyor is not initialized yet, so we pass a local closure instead of p.conveyor.AddManagedComponent
		// so the imposer can bypass calls to the conveyor
		addFn = func(c managed.Component) {
			p.conveyor.AddManagedComponent(c)
		}
		params := ImposedParams{
			CompartmentSetup:  appCfg,
			ConveyorRegistry:  p.convDeps,
			RegisterComponent: addFn,
		}
		p.imposeFn(&params)

		appCfg = params.CompartmentSetup
		p.convDeps = params.ConveyorRegistry
		addFn = params.RegisterComponent
	}

	var factoryFn conveyor.PulseEventFactoryFunc
	if appCfg.EventFactory != nil {
		factoryFn = appCfg.EventFactory.InputEvent
	}

	p.conveyor = conveyor.NewPulseConveyor(ctx, appCfg.ConveyorConfig, factoryFn, p.convDeps)

	if addFn == nil {
		addFn = p.conveyor.AddManagedComponent
	}

	if p.componentsFn != nil {
		p.componentsFn(ctx, inject, addFn)
	}

	if appCfg.ComponentSetupFn != nil {
		appCfg.ComponentSetupFn(ctx, inject, addFn)
	}

	if appCfg.EventFactory != nil {
		appCfg.EventFactory.SetupComponents(ctx, inject, addFn)
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

