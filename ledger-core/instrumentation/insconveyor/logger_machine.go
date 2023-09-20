package insconveyor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ConveyorLoggerFactory struct {
}

func (c ConveyorLoggerFactory) CreateStepLogger(ctx context.Context, _ smachine.StateMachine, traceID smachine.TracerID) smachine.StepLogger {
	ctxWithTrace, logger := inslogger.WithTraceField(ctx, traceID)
	return &ConveyorLogger{
		tracerCtx: ctxWithTrace,
		tracerID:  traceID,
		logger:    logger,
	}
}

type LogInternal struct {
	*log.Msg `txt:"internal"`

	Message   string `fmt:"internal - %s"`
	Component string `txt:"sm"`

	MachineID  string
	SlotID     smachine.SlotID
	SlotStepNo uint32
	CycleNo    uint32
	Error      error  `opt:""`
	Backtrace  string `opt:""`
}

func (ConveyorLoggerFactory) LogMachineInternal(slotMachineData smachine.SlotMachineData, msg string) {
	backtrace := ""
	level := log.WarnLevel
	if slotMachineData.Error != nil {
		level = log.ErrorLevel
		if st := throw.DeepestStackTraceOf(slotMachineData.Error); st != nil {
			st = throw.MinimizeStackTrace(st, convlog.StackMinimizePackage, true)
			backtrace = st.StackTraceAsText()
		}
	}

	global.Logger().Eventm(level, LogInternal{
		Message: msg,

		MachineID:  slotMachineData.StepNo.MachineID(),
		CycleNo:    slotMachineData.CycleNo,
		SlotID:     slotMachineData.StepNo.SlotID(),
		SlotStepNo: slotMachineData.StepNo.StepNo(),
		Error:      slotMachineData.Error,
		Backtrace:  backtrace,
	})
}

type LogCritical struct {
	*log.Msg `txt:"internal"`

	Message   string `fmt:"internal critical - %s"`
	Component string `txt:"sm"`

	MachineID  string
	SlotID     smachine.SlotID
	SlotStepNo uint32
	CycleNo    uint32
	Error      error  `opt:""`
	Backtrace  string `opt:""`
}

func (ConveyorLoggerFactory) LogMachineCritical(slotMachineData smachine.SlotMachineData, msg string) {
	backtrace := ""
	if slotMachineData.Error != nil {
		if st := throw.DeepestStackTraceOf(slotMachineData.Error); st != nil {
			st = throw.MinimizeStackTrace(st, convlog.StackMinimizePackage, true)
			backtrace = st.StackTraceAsText()
		}
	}

	global.Logger().Errorm(LogCritical{
		Message: msg,

		MachineID:  slotMachineData.StepNo.MachineID(),
		CycleNo:    slotMachineData.CycleNo,
		SlotID:     slotMachineData.StepNo.SlotID(),
		SlotStepNo: slotMachineData.StepNo.StepNo(),
		Error:      slotMachineData.Error,
		Backtrace:  backtrace,
	})
}

func (ConveyorLoggerFactory) LogStopping(*smachine.SlotMachine) {}
