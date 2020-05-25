// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotmachine

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	testUtilsCommon "github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/convlog"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/messagesender"
)

const (
	runTimeout = 30 * time.Second
)

type ControlledSlotMachine struct {
	t           *testing.T
	ctx         context.Context

	externalSignal synckit.VersionedSignal
	internalSignal synckit.VersionedSignal

	slotMachine *smachine.SlotMachine
	debugLogger *testUtilsCommon.DebugMachineLogger
	watchdog    *watchdog

	worker *Worker

	RunnerDescriptorCache *testutils.DescriptorCacheMockWrapper
	MachineManager        machine.Manager
	MessageSender         *messagesender.ServiceMockWrapper
}

func NewControlledSlotMachine(ctx context.Context, t *testing.T, suppressLogError bool) *ControlledSlotMachine {
	inslogger.SetTestOutput(t, suppressLogError)

	debugLogger := testUtilsCommon.NewDebugMachineLogger(convlog.MachineLogger{})
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    1e5,
		LogAdapterCalls:   true,
		SlotMachineLogger: debugLogger,
	}

	w := &ControlledSlotMachine{
		t:           t,
		ctx:         ctx,
		debugLogger: debugLogger,
	}
	w.slotMachine = smachine.NewSlotMachine(machineConfig,
		w.internalSignal.NextBroadcast,
		combineCallbacks(w.externalSignal.NextBroadcast, w.internalSignal.NextBroadcast),
		nil,
	)
	w.worker = NewWorker(w)

	return w
}

func combineCallbacks(mainFn, auxFn func()) func() {
	switch {
	case mainFn == nil:
		panic("illegal state")
	case auxFn == nil:
		return mainFn
	default:
		return func() {
			mainFn()
			auxFn()
		}
	}
}

func (c *ControlledSlotMachine) GetOccupiedSlotCount() int {
	return c.slotMachine.OccupiedSlotCount()
}

func (c *ControlledSlotMachine) Start() {
	if c.watchdog != nil {
		panic(throw.FailHere("double start"))
	}
	c.watchdog = newWatchdog(runTimeout)

	c.worker.Start()
}

func (c *ControlledSlotMachine) Step() testUtilsCommon.UpdateEvent {
	rv := c.debugLogger.GetEvent()
	c.debugLogger.Continue()
	return rv
}

func (c *ControlledSlotMachine) StepUntil(predicate func(event testUtilsCommon.UpdateEvent) bool) bool {
	for {
		event := c.Step()
		if event.IsEmpty() {
			return false
		}

		if predicate(event) {
			return true
		}
	}
}

func (c *ControlledSlotMachine) Stop() {
	c.watchdog.Stop()
	c.debugLogger.Stop()
	c.slotMachine.Stop()
}

func (c *ControlledSlotMachine) Migrate() {
	if !c.slotMachine.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}
}

func (c *ControlledSlotMachine) AddDependency(dep interface{}) {
	c.slotMachine.AddDependency(dep)
}

func (c *ControlledSlotMachine) AddInterfaceDependency(dep interface{}) {
	c.slotMachine.AddInterfaceDependency(dep)
}

func (c *ControlledSlotMachine) PrepareRunner(mc *minimock.Controller) {
	c.RunnerDescriptorCache = testutils.NewDescriptorsCacheMockWrapper(mc)
	c.MachineManager = machine.NewManager()

	runnerService := runner.NewService()
	runnerService.Manager = c.MachineManager
	runnerService.Cache = c.RunnerDescriptorCache.Mock()

	runnerAdapter := runner.CreateRunnerService(context.Background(), runnerService)
	c.slotMachine.AddDependency(runnerAdapter)
}

func (c *ControlledSlotMachine) PrepareMockedMessageSender(mc *minimock.Controller) {
	c.MessageSender = messagesender.NewServiceMockWrapper(mc)

	adapterMock := c.MessageSender.NewAdapterMock()

	var messageSender messageSenderAdapter.MessageSender = adapterMock.Mock()
	c.slotMachine.AddInterfaceDependency(&messageSender)
}

func (c *ControlledSlotMachine) AddStateMachine(ctx context.Context, sm smachine.StateMachine) *StateMachineWrapper {
	return NewStateMachineWrapper(sm, c.slotMachine.AddNew(ctx, sm, smachine.CreateDefaultValues{}))
}
