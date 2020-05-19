// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"fmt"
	"time"

	testWalletAPIStateMachine "github.com/insolar/assured-ledger/ledger-core/v2/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	flowDispatcher "github.com/insolar/assured-ledger/ledger-core/v2/insolar/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	runnerAdapter "github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	virtualStateMachine "github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

// TODO[bigbes] commented until panics will show description
// type errUnknownEvent struct {
// 	*log.Msg
//
// 	InputType interface{} `fmt:"%T"`
// }

func DefaultHandlersFactory(_ pulse.Number, _ pulse.Range, input conveyor.InputEvent) (pulse.Number, smachine.CreateFunc) {
	switch event := input.(type) {
	case *virtualStateMachine.DispatcherMessage:
		return handlers.FactoryMeta(event)
	case *testWalletAPIStateMachine.TestAPICall:
		return 0, testWalletAPIStateMachine.Handler(event)
	default:
		// TODO[bigbes] commented until panics will show description
		// panic(throw.E("unknown event type", errUnknownEvent{InputType: input}))
		panic(fmt.Sprintf("unknown event type %T", input))
	}
}

type Dispatcher struct {
	FlowDispatcher flowDispatcher.Dispatcher

	Conveyor       *conveyor.PulseConveyor
	ConveyorWorker virtualStateMachine.ConveyorWorker
	MachineLogger  smachine.SlotMachineLogger

	// Components
	Runner        *runner.DefaultService
	MessageSender messagesender.Service

	runnerAdapter        *runnerAdapter.ServiceAdapter
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

	lr.runnerAdapter = runner.CreateRunnerService(ctx, lr.Runner)
	lr.messageSenderAdapter = messageSenderAdapter.CreateMessageSendService(ctx, lr.MessageSender)

	lr.Conveyor.AddDependency(lr.runnerAdapter)
	lr.Conveyor.AddInterfaceDependency(&lr.messageSenderAdapter)

	var objectCatalog object.Catalog = object.NewLocalCatalog()
	lr.Conveyor.AddInterfaceDependency(&objectCatalog)

	lr.ConveyorWorker = virtualStateMachine.NewConveyorWorker()
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
