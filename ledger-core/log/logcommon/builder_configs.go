// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
)

type Config struct {
	BuildConfig

	BareOutput   BareOutput
	LoggerOutput LoggerOutput

	Metrics   *MetricsHelper
	MsgFormat logfmt.MsgFormatConfig

	ErrorFn func(error)
}

type BareOutput struct {
	Writer         io.Writer
	FlushFn        LogFlushFunc
	ProtectedClose bool
}

type BuildConfig struct {
	Output      OutputConfig
	Instruments InstrumentationConfig
}

type OutputConfig struct {
	BufferSize      int
	ParallelWriters int
	Format          LogFormat

	// allow buffer for regular events
	EnableRegularBuffer bool
}

func (v OutputConfig) CanReuseOutputFor(config OutputConfig) bool {
	return v.Format == config.Format &&
		(v.BufferSize > 0 || config.BufferSize <= 0)
}

type InstrumentationConfig struct {
	Recorder               LogMetricsRecorder
	MetricsMode            LogMetricsMode
	CallerMode             CallerFieldMode
	SkipFrameCountBaseline uint8
	SkipFrameCount         int8
}

func (v InstrumentationConfig) CanReuseOutputFor(config InstrumentationConfig) bool {
	vTWD := v.MetricsMode.HasWriteMetric()
	cTWD := config.MetricsMode.HasWriteMetric()

	if v.Recorder != config.Recorder {
		return !cTWD && !vTWD
	}

	return vTWD == cTWD || vTWD && !cTWD
}
