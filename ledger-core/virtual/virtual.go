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
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	virtualStateMachine "github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/virtual/token"
)

func DefaultHandlersFactory(_ pulse.Number, _ pulse.Range, input conveyor.InputEvent) (pulse.Number, smachine.CreateFunc) {
	switch event := input.(type) {
	case *virtualStateMachine.DispatcherMessage:
		return handlers.FactoryMeta(event)
	case *testWalletAPIStateMachine.TestAPICall:
		return 0, testWalletAPIStateMachine.Handler(event)
	default:
		panic(throw.E("unknown event type", struct {
			InputType interface{} `fmt:"%T"`
		}{InputType: input}))
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
	Runner        runner.Service
	MessageSender messagesender.Service
	TokenService  token.Service

	runnerAdapter        runner.ServiceAdapter
	messageSenderAdapter messageSenderAdapter.MessageSender

	stopFunc context.CancelFunc
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (lr *Dispatcher) Init(ctx context.Context) error {
	conveyorConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: virtualStateMachine.ConveyorLoggerFactory{},
		SlotAliasRegistry: &conveyor.GlobalAliases{},
		LogAdapterCalls:   true,
	}

	defaultHandlers := DefaultHandlersFactory

	machineConfig := conveyorConfig
	if lr.MachineLogger != nil {
		machineConfig.SlotMachineLogger = lr.MachineLogger
	}

	lr.Conveyor = conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: conveyorConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, defaultHandlers, nil)

	lr.runnerAdapter = lr.Runner.CreateAdapter(ctx)
	lr.messageSenderAdapter = messageSenderAdapter.CreateMessageSendService(ctx, lr.MessageSender)

	lr.Conveyor.AddInterfaceDependency(&lr.runnerAdapter)
	lr.Conveyor.AddInterfaceDependency(&lr.messageSenderAdapter)
	lr.Conveyor.AddInterfaceDependency(&lr.TokenService)

	var objectCatalog object.Catalog = object.NewLocalCatalog()
	lr.Conveyor.AddInterfaceDependency(&objectCatalog)

	lr.ConveyorWorker = virtualStateMachine.NewConveyorWorker(lr.CycleFn)
	lr.ConveyorWorker.AttachTo(lr.Conveyor)

	lr.FlowDispatcher = virtualStateMachine.NewConveyorDispatcher(lr.Conveyor)

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

func (lr *Dispatcher) AddInputExt(ctx context.Context, pulse pulse.Number, msg interface{}, createDefaults smachine.CreateDefaultValues) error {
	return lr.Conveyor.AddInputExt(ctx, pulse, msg, createDefaults)
}
