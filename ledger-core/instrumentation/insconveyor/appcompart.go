// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insconveyor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type AppCompartmentConfigFunc = func() conveyor.PulseConveyorConfig

type AppCompartment struct {
	FlowDispatcher beat.Dispatcher

	Conveyor       *conveyor.PulseConveyor
	ConveyorWorker ConveyorWorker
	MachineLogger  smachine.SlotMachineLogger
	CycleFn        conveyor.PulseConveyorCycleFunc

	MessageSender  messagesender.Service
	Affinity       affinity.Helper

	ConfigProvider AppCompartmentConfigFunc
	// EventlessSleep            time.Duration
	// FactoryLogContextOverride context.Context
}

func (p *AppCompartment) Init(ctx context.Context) error {
	ctx, _ = inslogger.WithField(ctx, "compartment", "??")

	cfg := p.ConfigProvider()

	p.Conveyor = conveyor.NewPulseConveyor(ctx, cfg, nil, nil)

	// defaultHandlers := DefaultHandlersFactory{}
	// defaultHandlers.AuthService = p.AuthenticationService
	// defaultHandlers.LogContextOverride = p.FactoryLogContextOverride

	// p.runnerAdapter = p.Runner.CreateAdapter(ctx)
	// p.messageSenderAdapter = messageSenderAdapter.CreateMessageSendService(ctx, p.MessageSender)
	//
	// p.Conveyor.AddInterfaceDependency(&p.runnerAdapter)
	// p.Conveyor.AddInterfaceDependency(&p.messageSenderAdapter)
	// p.Conveyor.AddInterfaceDependency(&p.AuthenticationService)
	//
	// var objectCatalog object.Catalog = object.NewLocalCatalog()
	// p.Conveyor.AddInterfaceDependency(&objectCatalog)
	//
	// runnerLimiter := tool.NewRunnerLimiter(p.MaxRunners)
	// p.Conveyor.AddDependency(runnerLimiter)

	p.ConveyorWorker = NewConveyorWorker(p.CycleFn)
	p.FlowDispatcher = NewConveyorDispatcher(ctx, p.Conveyor)

	return nil
}

func (p *AppCompartment) Start(context.Context) error {
	p.ConveyorWorker.AttachTo(p.Conveyor)
	return nil
}

func (p *AppCompartment) Stop(context.Context) error {
	p.ConveyorWorker.Stop()
	return nil
}

func (p *AppCompartment) AddInput(ctx context.Context, pn pulse.Number, event conveyor.InputEvent) error {
	return p.Conveyor.AddInput(ctx, pn, event)
}

func (p *AppCompartment) AddInputExt(pn pulse.Number, event conveyor.InputEvent, createDefaults smachine.CreateDefaultValues) error {
	return p.Conveyor.AddInputExt(pn, event, createDefaults)
}

func (p *AppCompartment) SetEventFactoryFunc(fn conveyor.PulseEventFactoryFunc) {
	p.Conveyor.SetFactoryFunc(fn)
}
