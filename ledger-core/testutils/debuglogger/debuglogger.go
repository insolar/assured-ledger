package debuglogger

import (
	"context"
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StepLogger = &DebugStepLogger{}

type UpdateEvent struct {
	notEmpty    bool
	SM          smachine.StateMachine
	Data        smachine.StepLoggerData
	Update      smachine.StepLoggerUpdateData
	CustomEvent interface{}
	AdapterID   smachine.AdapterID
	CallID      uint64
}

func (e UpdateEvent) IsEmpty() bool {
	return !e.notEmpty
}

type DebugStepLogger struct {
	smachine.StepLogger

	sm          smachine.StateMachine
	parent      *DebugMachineLogger
}

func (c DebugStepLogger) CanLogEvent(smachine.StepLoggerEvent, smachine.StepLogLevel) bool {
	return true
}

func (c DebugStepLogger) CreateAsyncLogger(context.Context, *smachine.StepLoggerData) (context.Context, smachine.StepLogger) {
	return c.GetLoggerContext(), c
}

func (c DebugStepLogger) LogInternal(data smachine.StepLoggerData, updateType string) {
	c.StepLogger.LogInternal(data, updateType)

	c.parent.events.send(UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		Update:   smachine.StepLoggerUpdateData{UpdateType: updateType},
	})

	c.parent.waitContinue()
}

func (c DebugStepLogger) LogUpdate(data smachine.StepLoggerData, update smachine.StepLoggerUpdateData) {
	c.StepLogger.LogUpdate(data, update)

	c.parent.events.send(UpdateEvent{
		notEmpty: true,
		SM:       c.sm,
		Data:     data,
		Update:   update,
	})

	c.parent.waitContinue()
}

func (c DebugStepLogger) CanLogTestEvent() bool {
	return true
}

func (c DebugStepLogger) LogTestEvent(data smachine.StepLoggerData, customEvent interface{}) {
	if c.StepLogger.CanLogTestEvent() {
		c.StepLogger.LogTestEvent(data, customEvent)
	}

	c.parent.events.send(UpdateEvent{
		notEmpty:    true,
		SM:          c.sm,
		Data:        data,
		CustomEvent: customEvent,
	})

	c.parent.waitContinue()
}

func (c DebugStepLogger) LogEvent(data smachine.StepLoggerData, customEvent interface{}, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogEvent(data, customEvent, fields)

	c.parent.events.send(UpdateEvent{
		notEmpty:    true,
		SM:          c.sm,
		Data:        data,
		CustomEvent: customEvent,
	})

	c.parent.waitContinue()
}

func (c DebugStepLogger) LogAdapter(data smachine.StepLoggerData, adapterID smachine.AdapterID, callID uint64, fields []logfmt.LogFieldMarshaller) {
	c.StepLogger.LogAdapter(data, adapterID, callID, fields)

	c.parent.events.send(UpdateEvent{
		notEmpty:  true,
		SM:        c.sm,
		Data:      data,
		AdapterID: adapterID,
		CallID:    callID,
	})

	c.parent.waitContinue()
}

type LoggerSlotPredicateFn func(smachine.StateMachine, smachine.TracerID) bool

type DebugMachineLogger struct {
	underlying   smachine.SlotMachineLogger
	events       updateChan
	continueStep synckit.ClosableSignalChannel
	abort        atomickit.OnceFlag
}

func (p *DebugMachineLogger) CreateStepLogger(ctx context.Context, sm smachine.StateMachine, traceID smachine.TracerID) smachine.StepLogger {
	underlying := p.underlying.CreateStepLogger(ctx, sm, traceID)

	return DebugStepLogger{
		StepLogger: underlying,
		sm:         sm,
		parent:     p,
	}
}

func (p *DebugMachineLogger) LogMachineInternal(slotMachineData smachine.SlotMachineData, msg string) {
	p.underlying.LogMachineInternal(slotMachineData, msg)
}

func (p *DebugMachineLogger) LogMachineCritical(slotMachineData smachine.SlotMachineData, msg string) {
	p.underlying.LogMachineCritical(slotMachineData, msg)
}

func (p *DebugMachineLogger) LogStopping(m *smachine.SlotMachine) {
	p.underlying.LogStopping(m)
	p.continueAll()
}

func (p *DebugMachineLogger) continueAll() {
	if p.continueStep != nil {
		select {
		case _, ok := <- p.continueStep:
			if !ok {
				return
			}
		default:
		}
		close(p.continueStep)
	}
}

func (p *DebugMachineLogger) GetEvent() (ev UpdateEvent) {
	 ev, _ = p.events.receive()
	 return
}

func (p *DebugMachineLogger) EventChan() <-chan UpdateEvent {
	return p.events.events
}

func (p *DebugMachineLogger) Continue() {
	if p.continueStep != nil {
		p.continueStep <- struct{}{}
	}
}

func (p *DebugMachineLogger) Abort() {
	if p.abort.Set() {
		p.continueAll()
	}
}

func (p *DebugMachineLogger) FlushEvents(flushDone synckit.SignalChannel, closeEvents bool) {
	for {
		select {
		case _, ok := <- p.events.events:
			if !ok {
				return
			}
		case <-flushDone:
			if closeEvents {
				p.events.close()
			}
			return
		}
	}
}

func (p *DebugMachineLogger) waitContinue() {
	if p.abort.IsSet() {
		runtime.Goexit()
	}
	if p.continueStep == nil {
		return
	}
	<- p.continueStep
	if p.abort.IsSet() {
		runtime.Goexit()
	}
}

func NewDebugMachineLogger(underlying smachine.SlotMachineLogger) *DebugMachineLogger {
	return &DebugMachineLogger{
		underlying:    underlying,
		events:        updateChan{ events: make(chan UpdateEvent, 1) },
		continueStep:  make(chan struct{}),
	}
}

func NewDebugMachineLoggerNoBlock(underlying smachine.SlotMachineLogger, eventBufLimit int) *DebugMachineLogger {
	if eventBufLimit <= 0 {
		panic(throw.IllegalValue())
	}
	return &DebugMachineLogger{
		underlying:    underlying,
		events:        updateChan{ events: make(chan UpdateEvent, eventBufLimit) },
	}
}
