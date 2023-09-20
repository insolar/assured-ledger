package inslogger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
)

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

	plc.OutputType, err = ParseOutput(cfg.OutputType)
	if err != nil {
		return
	}
	plc.OutputParam = cfg.OutputParams

	plc.Output.Format, err = ParseFormat(cfg.Formatter)
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

func ParseFormat(formatStr string) (logcommon.LogFormat, error) {
	return ParseFormatDef(formatStr, logcommon.TextFormat)
}

func ParseFormatDef(formatStr string, defValue logcommon.LogFormat) (logcommon.LogFormat, error) {
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

func ParseOutput(outputStr string) (logoutput.LogOutput, error) {
	return ParseOutputDef(outputStr, logoutput.StdErrOutput)
}

func ParseOutputDef(outputStr string, defValue logoutput.LogOutput) (logoutput.LogOutput, error) {
	switch strings.ToLower(outputStr) {
	case "", "default":
		return defValue, nil
	case logoutput.StdErrOutput.String():
		return logoutput.StdErrOutput, nil
	case logoutput.SysLogOutput.String():
		return logoutput.SysLogOutput, nil
	case logoutput.FileOutput.String():
		return logoutput.FileOutput, nil
	}
	return defValue, fmt.Errorf("unknown Output: '%s', replaced with '%s'", outputStr, defValue)
}
