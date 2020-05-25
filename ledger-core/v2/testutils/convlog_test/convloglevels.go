// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package convlog_test

import (
	"fmt"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gotest.tools/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type StepLoggerSuite struct {
	suite.Suite
	LoggerFn func(log.Logger) smachine.StepLogger
	Levels []LevelCase
}

type LevelCase struct{
	Event smachine.StepLoggerEvent
	LevelNorm, LevelElev, LevelErr log.Level
}

func (s StepLoggerSuite) TestLevels() {
	if len(s.Levels) == 0 {
		s.T().FailNow()
	}
}

func (s StepLoggerSuite) TestCanLogEvent() {
	for level := log.Disabled; level < log.NoLevel; level++ {
		for _, lc := range s.Levels {
			levelCase := lc
			for _, vc := range []struct { smachine.StepLogLevel; log.Level }{
				{smachine.StepLogLevelDefault, levelCase.LevelNorm},
				{smachine.StepLogLevelElevated, levelCase.LevelElev},
				{smachine.StepLogLevelTracing, levelCase.LevelElev},
				{smachine.StepLogLevelError, levelCase.LevelErr},
			} {
				v := vc
				s.Run(fmt.Sprintf("%v/%v/%v", level, levelCase.Event, v.StepLogLevel), func() {
					t := s.T()
					logger := logcommon.NewEmbeddedLoggerMock(s.T())
					stepLogger := s.LoggerFn(log.WrapEmbeddedLogger(logger))
					allowed := v.Level >= level
					logger.IsMock.Return(allowed)
					assert.Equal(t, allowed, stepLogger.CanLogEvent(levelCase.Event, v.StepLogLevel))
					logger.MinimockIsDone()
				})
			}
		}
	}
}

func (s StepLoggerSuite) TestLogEvent() {
	for level := log.DebugLevel; level < log.FatalLevel; level++ {
		for _, lc := range s.Levels {
			levelCase := lc
			for _, vc := range []struct { smachine.StepLogLevel; log.Level }{
				{smachine.StepLogLevelDefault, levelCase.LevelNorm},
				{smachine.StepLogLevelElevated, levelCase.LevelElev},
				{smachine.StepLogLevelTracing, levelCase.LevelElev},
				{smachine.StepLogLevelError, levelCase.LevelErr},
			} {
				if vc.Level < level {
					continue
				}

				v := vc
				s.Run(fmt.Sprintf("%v/%v/%v", level, levelCase.Event, v.StepLogLevel), func() {
					t := s.T()
					data := smachine.StepLoggerData{EventType: levelCase.Event}

					for {
						logger := logcommon.NewEmbeddedLoggerMock(s.T())
						stepLogger := s.LoggerFn(log.WrapEmbeddedLogger(logger))
						logger.IsMock.Return(true)
						logger.NewEventFmtMock.Return(func(string, []interface{}) {})
						logger.NewEventStructMock.Return(func(interface{}, []logfmt.LogFieldMarshaller) {})
						logger.NewEventMock.Return(func([]interface{}) {})
						logger.FieldsOfMock.Return(logfmt.LogField{Name: "stub", Value: "stub"})
						logger.EmbeddedFlushMock.Return()

						switch levelCase.Event {
						case smachine.StepLoggerUpdate, smachine.StepLoggerMigrate:
							stepLogger.LogUpdate(data, smachine.StepLoggerUpdateData{UpdateType: "test"})
						case smachine.StepLoggerInternal:
							stepLogger.LogInternal(data, "")
						case smachine.StepLoggerAdapterCall:
							stepLogger.LogAdapter(data, "test", 0, nil)
						case smachine.StepLoggerTrace, smachine.StepLoggerActiveTrace, smachine.StepLoggerWarn, smachine.StepLoggerError, smachine.StepLoggerFatal:
							stepLogger.LogEvent(data, "test", nil)
						default:
							t.FailNow()
						}
						logger.MinimockIsDone()

						callCount := logger.NewEventAfterCounter() + logger.NewEventFmtAfterCounter() + logger.NewEventStructAfterCounter()
						require.Equal(t, 1, int(callCount))

						if data.Error != nil || v.StepLogLevel == smachine.StepLogLevelError {
							break
						}
						data.Error = throw.New("test")
					}
				})
			}
		}
	}
}
