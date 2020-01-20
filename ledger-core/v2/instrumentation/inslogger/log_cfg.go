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

package inslogger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
)

const defaultLogFormat = logcommon.TextFormat
const defaultLogOutput = logoutput.StdErrOutput

type ParsedLogConfig struct {
	OutputType logoutput.LogOutput
	LogLevel   log.Level

	OutputParam string

	Output      logcommon.OutputConfig
	Instruments logcommon.InstrumentationConfig

	SkipFrameBaselineAdjustment int8
}

const defaultLowLatencyBufferSize = 100

func DefaultLoggerSettings() ParsedLogConfig {
	r := ParsedLogConfig{}
	r.Instruments.MetricsMode = logcommon.LogMetricsEventCount |
		logcommon.LogMetricsWriteDelayReport |
		logcommon.LogMetricsWriteDelayField |
		logcommon.LogMetricsTimestamp

	r.Instruments.CallerMode = logcommon.CallerField
	return r
}

func ParseLogConfig(cfg configuration.Log) (plc ParsedLogConfig, err error) {
	return ParseLogConfigWithDefaults(cfg, DefaultLoggerSettings())
}

func ParseLogConfigWithDefaults(cfg configuration.Log, defaults ParsedLogConfig) (plc ParsedLogConfig, err error) {
	plc = defaults

	plc.OutputType, err = ParseOutput(cfg.OutputType, defaultLogOutput)
	if err != nil {
		return
	}
	plc.OutputParam = cfg.OutputParams

	plc.Output.Format, err = ParseFormat(cfg.Formatter, defaultLogFormat)
	if err != nil {
		return
	}

	plc.LogLevel, err = log.ParseLevel(cfg.Level)
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
	case logcommon.JsonFormat.String():
		return logcommon.JsonFormat, nil
	}
	return defValue, fmt.Errorf("unknown Format: '%s', replaced with '%s'", formatStr, defValue)
}

func ParseOutput(outputStr string, defValue logoutput.LogOutput) (logoutput.LogOutput, error) {
	switch strings.ToLower(outputStr) {
	case "", "default":
		return defValue, nil
	case logoutput.StdErrOutput.String():
		return logoutput.StdErrOutput, nil
	case logoutput.SysLogOutput.String():
		return logoutput.SysLogOutput, nil
	}
	return defValue, fmt.Errorf("unknown Output: '%s', replaced with '%s'", outputStr, defValue)
}
