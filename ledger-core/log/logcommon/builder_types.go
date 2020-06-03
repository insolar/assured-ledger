// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"io"
	"time"
)

type Factory interface {
	PrepareBareOutput(output BareOutput, metrics *MetricsHelper, config BuildConfig) (io.Writer, error)
	CreateNewLogger(params NewLoggerParams) (EmbeddedLogger, error)
	CanReuseMsgBuffer() bool
}

type Template interface {
	Factory
	GetTemplateConfig() Config
	GetTemplateLevel() Level
	// NB! Must ignore RequiresLowLatency flag
	CopyTemplateLogger(CopyLoggerParams) EmbeddedLogger
}

type DynFieldFunc func() interface{}
type DynFieldMap map[string]DynFieldFunc

type LogMetricsRecorder interface {
	RecordLogEvent(level Level)
	RecordLogWrite(level Level)
	RecordLogDelay(level Level, d time.Duration)
}

type CallerFieldMode uint8

const (
	NoCallerField CallerFieldMode = iota
	CallerField
	CallerFieldWithFuncName
)

type LogMetricsMode uint8

const NoLogMetrics LogMetricsMode = 0
const (
	// Logger will report every event to metrics
	LogMetricsEventCount LogMetricsMode = 1 << iota
	// Logger will report to metrics a write duration (time since an event was created till it was directed to the output)
	LogMetricsWriteDelayReport
	// Logger will add a write duration field into to the output
	LogMetricsWriteDelayField
	// Logger will add a timestamp to every event
	LogMetricsTimestamp
	// No effect on logger. Indicates that WithMetrics should replace the mode, instead of adding it.
	LogMetricsResetMode
)

func (v LogMetricsMode) HasWriteMetric() bool {
	return v&(LogMetricsWriteDelayReport|LogMetricsWriteDelayField) != 0
}

type LogFormat string

const (
	TextFormat LogFormat = "text"
	JSONFormat LogFormat = "json"
	PbufFormat LogFormat = "pbuf"
)

func (l LogFormat) String() string {
	return string(l)
}

type FactoryRequirementFlags uint8

const (
	RequiresLowLatency FactoryRequirementFlags = 1 << iota
	RequiresParentCtxFields
	RequiresParentDynFields
)

type CopyLoggerParams struct {
	Reqs            FactoryRequirementFlags
	Level           Level
	AppendFields    map[string]interface{}
	AppendDynFields DynFieldMap
}

type NewLoggerParams struct {
	Reqs      FactoryRequirementFlags
	Level     Level
	Fields    map[string]interface{}
	DynFields DynFieldMap

	Config Config
}

//type GlobalLogAdapterFactory interface {
//	GetGlobalLogAdapter() GlobalLogAdapter
//}

type GlobalLogAdapter interface {
	SetGlobalLoggerFilter(level Level)
	GetGlobalLoggerFilter() Level
}
