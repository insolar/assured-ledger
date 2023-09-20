package smachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

type StepLoggerStub struct {
	TracerID TracerID
}

func (StepLoggerStub) CanLogEvent(StepLoggerEvent, StepLogLevel) bool {
	return false
}

func (StepLoggerStub) CanLogTestEvent() bool {
	return false
}

func (StepLoggerStub) LogUpdate(StepLoggerData, StepLoggerUpdateData)                            {}
func (StepLoggerStub) LogInternal(StepLoggerData, string)                                        {}
func (StepLoggerStub) LogTestEvent(StepLoggerData, interface{}) 								 {}
func (StepLoggerStub) LogEvent(StepLoggerData, interface{}, []logfmt.LogFieldMarshaller)         {}
func (StepLoggerStub) LogAdapter(StepLoggerData, AdapterID, uint64, []logfmt.LogFieldMarshaller) {}

func (StepLoggerStub) CreateAsyncLogger(ctx context.Context, data *StepLoggerData) (context.Context, StepLogger) {
	return ctx, nil
}

func (v StepLoggerStub) GetTracerID() TracerID {
	return v.TracerID
}

func (v StepLoggerStub) GetLoggerContext() context.Context {
	return nil
}

type fixedSlotLogger struct {
	logger StepLogger
	level  StepLogLevel
	data   StepLoggerData
}

func (v fixedSlotLogger) getStepLogger() (StepLogger, StepLogLevel, uint32) {
	if step, ok := v.data.StepNo.SlotLink.GetStepLink(); ok {
		return v.logger, v.level, step.StepNo()
	}
	return v.logger, v.level, 0
}

func (v fixedSlotLogger) getStepLoggerData() StepLoggerData {
	return v.data
}
