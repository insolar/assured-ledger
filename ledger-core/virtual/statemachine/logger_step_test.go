// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/convlog_test"
)

var levelCases = []convlog_test.LevelCase{
	{smachine.StepLoggerUpdate, log.DebugLevel, log.InfoLevel, log.ErrorLevel},
	{smachine.StepLoggerMigrate, log.DebugLevel, log.InfoLevel, log.ErrorLevel},
	{smachine.StepLoggerInternal, log.DebugLevel, log.InfoLevel, log.ErrorLevel},
	{smachine.StepLoggerAdapterCall, log.DebugLevel, log.InfoLevel, log.ErrorLevel},
	{smachine.StepLoggerTrace, log.DebugLevel, log.InfoLevel, log.ErrorLevel},
	{smachine.StepLoggerActiveTrace, log.InfoLevel, log.InfoLevel, log.ErrorLevel},
	{smachine.StepLoggerWarn, log.WarnLevel, log.WarnLevel, log.ErrorLevel},
	{smachine.StepLoggerError, log.ErrorLevel, log.ErrorLevel, log.ErrorLevel},
	{smachine.StepLoggerFatal, log.FatalLevel, log.FatalLevel, log.FatalLevel},
}

func TestLevels(t *testing.T) {
	suite.Run(t, &convlog_test.StepLoggerSuite{
		Levels: levelCases,
		LoggerFn: func(logger log.Logger) smachine.StepLogger {
			return ConveyorLogger{context.Background(), "", logger}
		}})
}
