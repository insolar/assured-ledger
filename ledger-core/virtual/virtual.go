// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"time"

	testWalletAPIStateMachine "github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	flowDispatcher "github.com/insolar/assured-ledger/ledger-core/insolar/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	virtualStateMachine "github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
)

type DefaultHandlersFactory struct {
	handlers.FactoryMeta
}

func (f DefaultHandlersFactory) Classify(ctx context.Context, input conveyor.InputEvent, ic conveyor.InputContext) (conveyor.InputSetup, error) {
	switch event := input.(type) {
	case *virtualStateMachine.DispatcherMessage:
		if ic.PulseRange == nil {
			return conveyor.InputSetup{},
			throw.E("event is too old", struct {
				PN        pulse.Number
				InputType interface{} `fmt:"%T"`
			}{ic.PulseNumber, input})
		}

		targetPN, createFn, err := f.Process(ctx, event, ic.PulseRange)
		return conveyor.InputSetup{
			TargetPulse: targetPN,
			CreateFn:    createFn,
		}, err

	case *testWalletAPIStateMachine.TestAPICall:
		return conveyor.InputSetup{
			CreateFn: testWalletAPIStateMachine.Handler(event),
		}, nil
	default:
		panic(throw.E("unknown event type", struct {
			PN        pulse.Number
			InputType interface{} `fmt:"%T"`
		}{ic.PulseNumber, input}))
	}
}

type Dispatcher struct {
	FlowDispatcher flowDispatcher.Dispatcher

	Conveyor       *conveyor.PulseConveyor
	ConveyorWorker virtualStateMachine.ConveyorWorker
	MachineLogger  smachine.SlotMachineLogger

	// CycleFn is called after every scan cycle done by conveyor worker
	CycleFn conveyor.PulseConveyorCycleFunc

	// Components
	Runner                runner.Service
	MessageSender         messagesender.Service
	AuthenticationService authentication.Service
	Affinity              jet.AffinityHelper

	EventlessSleep time.Duration
	FactoryLogContextOverride context.Context

	runnerAdapter        runner.ServiceAdapter
	messageSenderAdapter messageSenderAdapter.MessageSender

	stopFunc                  context.CancelFunc
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (lr *Dispatcher) Init(ctx context.Context) error {
	ctx, _ = inslogger.WithField(ctx, "component", "sm")

	conveyorConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: virtualStateMachine.ConveyorLoggerFactory{},
		SlotAliasRegistry: &conveyor.GlobalAliases{},
		LogAdapterCalls:   true,
	}

	machineConfig := conveyorConfig
	if lr.MachineLogger != nil {
		machineConfig.SlotMachineLogger = lr.MachineLogger
	}

	switch {
	case lr.EventlessSleep == 0:
		lr.EventlessSleep = 100 * time.Millisecond
	case lr.EventlessSleep < 0:
		lr.EventlessSleep = 0
	}

	lr.Conveyor = conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: conveyorConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        lr.EventlessSleep,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, nil, nil)

	defaultHandlers := DefaultHandlersFactory{}
	defaultHandlers.AuthService = lr.AuthenticationService
	defaultHandlers.LogContextOverride = lr.FactoryLogContextOverride

	lr.Conveyor.SetFactoryFunc(defaultHandlers.Classify)

	lr.runnerAdapter = lr.Runner.CreateAdapter(ctx)
	lr.messageSenderAdapter = messageSenderAdapter.CreateMessageSendService(ctx, lr.MessageSender)

	lr.Conveyor.AddInterfaceDependency(&lr.runnerAdapter)
	lr.Conveyor.AddInterfaceDependency(&lr.messageSenderAdapter)
	lr.Conveyor.AddInterfaceDependency(&lr.AuthenticationService)

	var objectCatalog object.Catalog = object.NewLocalCatalog()
	lr.Conveyor.AddInterfaceDependency(&objectCatalog)

	lr.ConveyorWorker = virtualStateMachine.NewConveyorWorker(lr.CycleFn)
	lr.ConveyorWorker.AttachTo(lr.Conveyor)

	lr.FlowDispatcher = virtualStateMachine.NewConveyorDispatcher(ctx, lr.Conveyor)

	return nil
}

func (lr *Dispatcher) Start(_ context.Context) error {
	return nil
}

func (lr *Dispatcher) Stop(_ context.Context) error {
	lr.ConveyorWorker.Stop()
	lr.stopFunc()

	return nil
}

func (lr *Dispatcher) AddInput(ctx context.Context, pulse pulse.Number, msg interface{}) error {
	return lr.Conveyor.AddInput(ctx, pulse, msg)
}

func (lr *Dispatcher) AddInputExt(pulse pulse.Number, msg interface{}, createDefaults smachine.CreateDefaultValues) error {
	return lr.Conveyor.AddInputExt(pulse, msg, createDefaults)
}
