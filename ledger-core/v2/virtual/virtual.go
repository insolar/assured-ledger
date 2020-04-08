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
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow/dispatcher"
	lrCommon "github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/common"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

type Virtual interface {
	component.Initer
	component.Starter
	component.Stopper
}

type virtual struct {
	FlowDispatcher dispatcher.Dispatcher

	Conveyor       *conveyor.PulseConveyor
	ConveyorWorker lrCommon.ConveyorWorker

	Cfg *configuration.LogicRunner
}

func NewVirtual() (Virtual, error) {
	return &virtual{}, nil
}

func (lr *virtual) Init(_ context.Context) error {
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

	lr.ConveyorWorker = lrCommon.NewConveyorWorker()
	lr.ConveyorWorker.AttachTo(lr.Conveyor)

	lr.FlowDispatcher = lrCommon.NewConveyorDispatcher(lr.Conveyor)

	return nil
}

func (lr *virtual) Start(_ context.Context) error {
	return nil
}

func (lr *virtual) Stop(_ context.Context) error {
	lr.ConveyorWorker.Stop()

	return nil
}
