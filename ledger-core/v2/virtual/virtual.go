// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtual

import (
	"context"
	"time"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	flowDispatcher "github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

type Dispatcher interface {
	component.Initer
	component.Starter
	component.Stopper
}

type dispatcher struct {
	FlowDispatcher flowDispatcher.Dispatcher

	Conveyor       *conveyor.PulseConveyor
	ConveyorWorker statemachine.ConveyorWorker

	Cfg *configuration.LogicRunner
}

func NewDispatcher() (Dispatcher, error) {
	return &dispatcher{}, nil
}

func (lr *dispatcher) Init(_ context.Context) error {
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: statemachine.ConveyorLoggerFactory{},
	}

	defaultHandlers := statemachine.DefaultHandlersFactory

	lr.Conveyor = conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, defaultHandlers, nil)

	lr.ConveyorWorker = statemachine.NewConveyorWorker()
	lr.ConveyorWorker.AttachTo(lr.Conveyor)

	lr.FlowDispatcher = statemachine.NewConveyorDispatcher(lr.Conveyor)

	return nil
}

func (lr *dispatcher) Start(_ context.Context) error {
	return nil
}

func (lr *dispatcher) Stop(_ context.Context) error {
	lr.ConveyorWorker.Stop()

	return nil
}
