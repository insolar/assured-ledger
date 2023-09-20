package insconveyor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/testutils/convlog_test"
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
