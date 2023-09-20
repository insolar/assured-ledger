package slotdebugger

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	testUtilsCommon "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	runTimeout = 30 * time.Second
)

type StepController struct {
	ctx context.Context

	externalSignal synckit.VersionedSignal
	internalSignal synckit.VersionedSignal

	SlotMachine   *smachine.SlotMachine
	PulseSlot     conveyor.PulseSlot
	debugLogger   *debuglogger.DebugMachineLogger
	stepIsWaiting bool
	watchdog      *watchdog

	worker        *worker
	MessageSender *messagesender.ServiceMockWrapper
}

func NewWithErrorFilter(ctx context.Context, t *testing.T, filterFn logcommon.ErrorFilterFunc) *StepController {
	controller := newController(ctx, t, filterFn)
	controller.preparePresentPulseSlot()
	return controller
}

// deprecated
func NewWithIgnoreAllErrors(ctx context.Context, t *testing.T) *StepController {
	return NewWithErrorFilter(ctx, t, func(string) bool { return false })
}

func New(ctx context.Context, t *testing.T) *StepController {
	return NewWithErrorFilter(ctx, t, nil)
}

func NewPast(ctx context.Context, t *testing.T) *StepController {
	controller := newController(ctx, t, nil)
	controller.preparePastPulseSlot()
	return controller
}

func newController(ctx context.Context, t *testing.T, filterFn logcommon.ErrorFilterFunc) *StepController {
	instestlogger.SetTestOutputWithErrorFilter(t, filterFn)

	var machineLogger smachine.SlotMachineLogger
	if convlog.UseTextConvLog() {
		machineLogger = convlog.MachineLogger{}
	} else {
		machineLogger = insconveyor.ConveyorLoggerFactory{}
	}

	debugLogger := debuglogger.NewDebugMachineLogger(machineLogger)
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
		ctx:         ctx,
		debugLogger: debugLogger,
	}
	w.SlotMachine = smachine.NewSlotMachine(machineConfig,
		w.internalSignal.NextBroadcast,
		combineCallbacks(w.externalSignal.NextBroadcast, w.internalSignal.NextBroadcast),
		nil,
	)
	w.worker = newWorker(w)

	return w
}

func (c *StepController) preparePresentPulseSlot() {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	c.PulseSlot = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
	c.SlotMachine.AddDependency(&c.PulseSlot)
}

func (c *StepController) preparePastPulseSlot() {
	pd := pulse.NewFirstPulsarData(10, longbits.Bits256{})
	c.PulseSlot = conveyor.NewPastPulseSlot(nil, pd.AsRange())
	c.SlotMachine.AddDependency(&c.PulseSlot)
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

func (c *StepController) NextStep() debuglogger.UpdateEvent {
	if c.stepIsWaiting {
		c.debugLogger.Continue()
	}
	rv := c.debugLogger.GetEvent()
	c.stepIsWaiting = !rv.IsEmpty()
	return rv
}

func (c *StepController) RunTil(predicate func(event debuglogger.UpdateEvent) bool) {
	for {
		switch event := c.NextStep(); {
		case event.IsEmpty():
			panic(throw.IllegalState())
		case predicate(event):
			return
		}
	}
}

func (c *StepController) Continue() {
	c.debugLogger.Continue()
}

func (c *StepController) Stop() {
	c.watchdog.Stop()

	c.debugLogger.Abort()
	c.SlotMachine.Stop()

	c.debugLogger.FlushEvents(c.worker.finishedSignal(), true)
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
	link, _ := c.SlotMachine.AddNew(ctx, sm, smachine.CreateDefaultValues{})
	return NewStateMachineHelper(link)
}
