// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type StepLoggerStub struct {
	TracerID TracerID
}

func (StepLoggerStub) CanLogEvent(StepLoggerEvent, StepLogLevel) bool {
	return false
}

func (StepLoggerStub) LogUpdate(StepLoggerData, StepLoggerUpdateData)                            {}
func (StepLoggerStub) LogInternal(StepLoggerData, string)                                        {}
func (StepLoggerStub) LogEvent(StepLoggerData, interface{}, []logfmt.LogFieldMarshaller)         {}
func (StepLoggerStub) LogAdapter(StepLoggerData, AdapterID, uint64, []logfmt.LogFieldMarshaller) {}

func (StepLoggerStub) CreateAsyncLogger(ctx context.Context, data *StepLoggerData) (context.Context, StepLogger) {
	return ctx, nil
}

func (v StepLoggerStub) GetTracerID() TracerID {
	return v.TracerID
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
	return nil, 0, 0
}

func (v fixedSlotLogger) getStepLoggerData() StepLoggerData {
	return v.data
}
