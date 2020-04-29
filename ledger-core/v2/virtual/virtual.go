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
	flowDispatcher "github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	runnerAdapter "github.com/insolar/assured-ledger/ledger-core/v2/runner/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/request"
	virtualStateMachine "github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

// TODO[bigbes] commented until panics will show description
// type errUnknownEvent struct {
// 	*log.Msg
//
// 	InputType interface{} `fmt:"%T"`
// }

func DefaultHandlersFactory(_ pulse.Number, input conveyor.InputEvent) smachine.CreateFunc {
	switch event := input.(type) {
	case *virtualStateMachine.DispatcherMessage:
		return request.HandlerFactoryMeta(event)
	case *testWalletAPIStateMachine.TestAPICall:
		return testWalletAPIStateMachine.Handler(event)
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

	// Components
	Runner        runner.Service
	MessageSender messagesender.Service

	runnerAdapter        *runnerAdapter.Runner
	messageSenderAdapter *messageSenderAdapter.MessageSender
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (lr *Dispatcher) Init(ctx context.Context, loggerOverride smachine.SlotMachineLogger) error {
	conveyorConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: virtualStateMachine.ConveyorLoggerFactory{},
		SlotAliasRegistry: &conveyor.GlobalAliases{},
	}

	machineConfig := conveyorConfig
	if loggerOverride != nil {
		machineConfig.SlotMachineLogger = loggerOverride
	}

	defaultHandlers := DefaultHandlersFactory

	lr.Conveyor = conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: conveyorConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, defaultHandlers, nil)

	lr.runnerAdapter = runnerAdapter.CreateRunnerServiceAdapter(ctx, lr.Runner)
	lr.messageSenderAdapter = messageSenderAdapter.CreateMessageSendService(ctx, lr.MessageSender)

	lr.Conveyor.AddDependency(lr.runnerAdapter)
	lr.Conveyor.AddDependency(lr.messageSenderAdapter)

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

	return nil
}

func (lr *Dispatcher) AddInput(ctx context.Context, pulse pulse.Number, msg interface{}) error {
	return lr.Conveyor.AddInput(ctx, pulse, msg)
}
