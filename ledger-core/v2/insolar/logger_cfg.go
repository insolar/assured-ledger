//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package insolar

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
)

const TimestampFormat = "2006-01-02T15:04:05.000000000Z07:00"

const DefaultLogFormat = logcommon.TextFormat
const DefaultLogOutput = logadapter.StdErrOutput

type ParsedLogConfig struct {
	OutputType logadapter.LogOutput
	LogLevel   logcommon.LogLevel
	//GlobalLevel logcommon.LogLevel

	OutputParam string

	Output      logadapter.OutputConfig
	Instruments logadapter.InstrumentationConfig

	SkipFrameBaselineAdjustment int8
}

const defaultLowLatencyBufferSize = 100

func DefaultLoggerSettings() ParsedLogConfig {
	r := ParsedLogConfig{}
	r.Instruments.MetricsMode = logcommon.LogMetricsEventCount | logcommon.LogMetricsWriteDelayReport | logcommon.LogMetricsWriteDelayField
	r.Instruments.CallerMode = logcommon.CallerField
	return r
}

func ParseLogConfig(cfg configuration.Log) (plc ParsedLogConfig, err error) {
	return ParseLogConfigWithDefaults(cfg, DefaultLoggerSettings())
}

func ParseLogConfigWithDefaults(cfg configuration.Log, defaults ParsedLogConfig) (plc ParsedLogConfig, err error) {
	plc = defaults

	plc.OutputType, err = ParseOutput(cfg.OutputType, DefaultLogOutput)
	if err != nil {
		return
	}
	plc.OutputParam = cfg.OutputParams

	plc.Output.Format, err = ParseFormat(cfg.Formatter, DefaultLogFormat)
	if err != nil {
		return
	}

	plc.LogLevel, err = logcommon.ParseLevel(cfg.Level)
	if err != nil {
		return
	}

	if len(cfg.OutputParallelLimit) > 0 {
		plc.Output.ParallelWriters, err = strconv.Atoi(cfg.OutputParallelLimit)
		if err != nil {
			return
		}
	} else {
		plc.Output.ParallelWriters = 0
	}

	//plc.GlobalLevel, err = logcommon.ParseLevel(cfg.GlobalLevel)
	//if err != nil {
	//	plc.GlobalLevel = logcommon.NoLevel
	//}

	switch {
	case cfg.LLBufferSize < 0:
		// LL buffer is disabled
		plc.Output.BufferSize = cfg.BufferSize
	case cfg.LLBufferSize > 0:
		plc.Output.BufferSize = cfg.LLBufferSize
	default:
		plc.Output.BufferSize = defaultLowLatencyBufferSize
	}

	if plc.Output.BufferSize < cfg.BufferSize {
		plc.Output.BufferSize = cfg.BufferSize
	}
	plc.Output.EnableRegularBuffer = cfg.BufferSize > 0

	return plc, nil
}

func ParseFormat(formatStr string, defValue logcommon.LogFormat) (logcommon.LogFormat, error) {
	switch strings.ToLower(formatStr) {
	case "", "default":
		return defValue, nil
	case logcommon.TextFormat.String():
		return logcommon.TextFormat, nil
	case logcommon.JSONFormat.String():
		return logcommon.JSONFormat, nil
	}
	return defValue, fmt.Errorf("unknown Format: '%s', replaced with '%s'", formatStr, defValue)
}

func ParseOutput(outputStr string, defValue logadapter.LogOutput) (logadapter.LogOutput, error) {
	switch strings.ToLower(outputStr) {
	case "", "default":
		return defValue, nil
	case logadapter.StdErrOutput.String():
		return logadapter.StdErrOutput, nil
	case logadapter.SysLogOutput.String():
		return logadapter.SysLogOutput, nil
	}
	return defValue, fmt.Errorf("unknown Output: '%s', replaced with '%s'", outputStr, defValue)
}
