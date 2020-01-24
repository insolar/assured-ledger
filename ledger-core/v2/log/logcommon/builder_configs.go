//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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
