// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotdebugger

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	testUtilsCommon "github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/convlog"
)

const (
	runTimeout = 30 * time.Second
)

type StepController struct {
	t   *testing.T
	ctx context.Context

	externalSignal synckit.VersionedSignal
	internalSignal synckit.VersionedSignal

	SlotMachine   *smachine.SlotMachine
	debugLogger   *testUtilsCommon.DebugMachineLogger
	stepIsWaiting bool
	watchdog      *watchdog

	worker        *worker
	MessageSender *messagesender.ServiceMockWrapper
}

func New(ctx context.Context, t *testing.T, suppressLogError bool) *StepController {
	inslogger.SetTestOutput(t, suppressLogError)

	debugLogger := testUtilsCommon.NewDebugMachineLogger(convlog.MachineLogger{})
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    1e5,
		LogAdapterCalls:   true,
		SlotMachineLogger: debugLogger,
		SlotAliasRegistry: &conveyor.GlobalAliases{},
	}

	w := &StepController{
		t:           t,
		ctx:         ctx,
		debugLogger: debugLogger,
	}
	w.SlotMachine = smachine.NewSlotMachine(machineConfig,
		w.internalSignal.NextBroadcast,
		combineCallbacks(w.externalSignal.NextBroadcast, w.internalSignal.NextBroadcast),
		nil,
	)
	w.worker = newWorker(w)

	w.preparePulseSlot()

	return w
}
func (c *StepController) preparePulseSlot() {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	pulseSlot := conveyor.NewPresentPulseSlot(nil, pd.AsRange())
	c.SlotMachine.AddDependency(&pulseSlot)
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

func (c *StepController) GetOccupiedSlotCount() int {
	return c.SlotMachine.OccupiedSlotCount()
}

func (c *StepController) Start() {
	if !testUtilsCommon.IsLocalDebug() {
		if c.watchdog != nil {
			panic(throw.FailHere("double start"))
		}
		c.watchdog = newWatchdog(runTimeout)
	}

	c.worker.Start()
}

func (c *StepController) NextStep() testUtilsCommon.UpdateEvent {
	if c.stepIsWaiting {
		c.debugLogger.Continue()
	}
	rv := c.debugLogger.GetEvent()
	c.stepIsWaiting = !rv.IsEmpty()
	return rv
}

func (c *StepController) RunTil(predicate func(event testUtilsCommon.UpdateEvent) bool) {
	for {
		switch event := c.NextStep(); {
		case event.IsEmpty():
			panic(throw.IllegalState())
		case predicate(event):
			return
		}
	}
}

func (c *StepController) Stop() {
	c.watchdog.Stop()

	c.SlotMachine.Stop()
	c.externalSignal.NextBroadcast()
	c.debugLogger.Stop()

	c.debugLogger.FlushEvents(c.worker.finishedSignal())
}

func (c *StepController) Migrate() {
	if !c.SlotMachine.ScheduleCall(func(callContext smachine.MachineCallContext) {
		callContext.Migrate(nil)
	}, true) {
		panic(throw.IllegalState())
	}
}

func (c *StepController) AddDependency(dep interface{}) {
	c.SlotMachine.AddDependency(dep)
}

func (c *StepController) AddInterfaceDependency(dep interface{}) {
	c.SlotMachine.AddInterfaceDependency(dep)
}

func (c *StepController) InitEmptyMessageSender(mc *minimock.Controller) {
	c.MessageSender = messagesender.NewServiceMockWrapper(mc)

	adapterMock := c.MessageSender.NewAdapterMock()

	var messageSender messageSenderAdapter.MessageSender = adapterMock.Mock()

	c.SlotMachine.AddInterfaceDependency(&messageSender)
}

func (c *StepController) PrepareMockedMessageSender(mc *minimock.Controller) {
	c.MessageSender = messagesender.NewServiceMockWrapper(mc)

	adapterMock := c.MessageSender.NewAdapterMock().SetDefaultPrepareAsyncCall(c.ctx)

	var messageSender messageSenderAdapter.MessageSender = adapterMock.Mock()

	c.SlotMachine.AddInterfaceDependency(&messageSender)
}

func (c *StepController) AddStateMachine(ctx context.Context, sm smachine.StateMachine) StateMachineHelper {
	return NewStateMachineHelper(sm, c.SlotMachine.AddNew(ctx, sm, smachine.CreateDefaultValues{}))
}
