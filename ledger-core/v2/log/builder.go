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

package log

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logwriter"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/bpbuffer"
)

func NewBuilderWithTemplate(template logcommon.Template, level Level) LoggerBuilder {
	config := template.GetTemplateConfig()
	return LoggerBuilder{
		factory:     template,
		hasTemplate: true,
		level:       level,
		Config:      config,
	}
}

func NewBuilder(factory logcommon.Factory, config logcommon.Config, level Level) LoggerBuilder {
	return LoggerBuilder{
		factory: factory,
		level:   level,
		Config:  config,
	}
}

type LoggerBuilder struct {
	factory     logcommon.Factory
	hasTemplate bool

	level Level

	noFields    bool
	noDynFields bool

	fields    map[string]interface{}
	dynFields logcommon.DynFieldMap

	logcommon.Config
}

func (z LoggerBuilder) IsZero() bool {
	return z.factory == nil
}

func (z LoggerBuilder) GetOutput() io.Writer {
	return z.BareOutput.Writer
}

func (z LoggerBuilder) GetLogLevel() Level {
	return z.level
}

func (z LoggerBuilder) WithOutput(w io.Writer) LoggerBuilder {

	z.BareOutput = logcommon.BareOutput{Writer: w}
	switch ww := w.(type) {
	case interface{ Flush() error }:
		z.BareOutput.FlushFn = ww.Flush
	case interface{ Sync() error }:
		z.BareOutput.FlushFn = ww.Sync
	}

	return z
}

func (z LoggerBuilder) WithBuffer(bufferSize int, bufferForAll bool) LoggerBuilder {
	z.Output.BufferSize = bufferSize
	z.Output.EnableRegularBuffer = bufferForAll
	return z
}

func (z LoggerBuilder) WithLevel(level Level) LoggerBuilder {
	z.level = level
	return z
}

func (z LoggerBuilder) WithFormat(format logcommon.LogFormat) LoggerBuilder {
	z.Output.Format = format
	return z
}

func (z LoggerBuilder) WithCaller(mode logcommon.CallerFieldMode) LoggerBuilder {
	z.Instruments.CallerMode = mode
	return z
}

func (z LoggerBuilder) WithMetrics(mode logcommon.LogMetricsMode) LoggerBuilder {
	if mode&logcommon.LogMetricsResetMode != 0 {
		z.Instruments.MetricsMode = 0
		mode &^= logcommon.LogMetricsResetMode
	}
	z.Instruments.MetricsMode |= mode
	return z
}

func (z LoggerBuilder) WithMetricsRecorder(recorder logcommon.LogMetricsRecorder) LoggerBuilder {
	z.Instruments.Recorder = recorder
	return z
}

func (z LoggerBuilder) WithSkipFrameCount(skipFrameCount int) LoggerBuilder {
	if skipFrameCount < math.MinInt8 || skipFrameCount > math.MaxInt8 {
		panic("illegal value")
	}
	z.Instruments.SkipFrameCount = int8(skipFrameCount)
	return z
}

func (z LoggerBuilder) WithoutInheritedFields() LoggerBuilder {
	z.noFields = true
	z.noDynFields = true
	return z
}

func (z LoggerBuilder) WithoutInheritedDynFields() LoggerBuilder {
	z.noDynFields = true
	return z
}

func (z LoggerBuilder) WithFields(fields map[string]interface{}) LoggerBuilder {
	if z.fields == nil {
		z.fields = make(map[string]interface{}, len(fields))
	}
	for k, v := range fields {
		delete(z.dynFields, k)
		z.fields[k] = v
	}
	return z
}

func (z LoggerBuilder) WithField(k string, v interface{}) LoggerBuilder {
	if z.fields == nil {
		z.fields = make(map[string]interface{})
	}
	delete(z.dynFields, k)
	z.fields[k] = v
	return z
}

func (z LoggerBuilder) WithDynamicField(k string, fn logcommon.DynFieldFunc) LoggerBuilder {
	if fn == nil {
		panic("illegal value")
	}
	if z.dynFields == nil {
		z.dynFields = make(logcommon.DynFieldMap)
	}
	delete(z.fields, k)
	z.dynFields[k] = fn
	return z
}

func (z LoggerBuilder) Build() (Logger, error) {
	return z.build(false)
}

func (z LoggerBuilder) BuildLowLatency() (Logger, error) {
	return z.build(true)
}

func (z LoggerBuilder) build(needsLowLatency bool) (Logger, error) {
	if el, err := z.buildEmbedded(needsLowLatency); err != nil {
		return nil, err
	} else {
		return WrapEmbeddedLogger(el), nil
	}
}

func (z LoggerBuilder) buildEmbedded(needsLowLatency bool) (logcommon.EmbeddedLogger, error) {
	var metrics *logcommon.MetricsHelper

	if z.Config.Instruments.MetricsMode != logcommon.NoLogMetrics {
		metrics = logcommon.NewMetricsHelper(z.Config.Instruments.Recorder)
	}

	reqs := logcommon.RequiresParentCtxFields | logcommon.RequiresParentDynFields
	switch {
	case z.noFields:
		reqs &^= logcommon.RequiresParentCtxFields | logcommon.RequiresParentDynFields
	case z.noDynFields:
		reqs &^= logcommon.RequiresParentDynFields
	}
	if needsLowLatency {
		reqs |= logcommon.RequiresLowLatency
	}

	var output logcommon.LoggerOutput

	switch {
	case z.BareOutput.Writer == nil:
		return nil, errors.New("output is nil")
	case z.hasTemplate:
		template := z.factory.(logcommon.Template)
		origConfig := template.GetTemplateConfig()

		sameBareOutput := false
		switch {
		case z.BareOutput.Writer == origConfig.LoggerOutput: // users can be crazy
			fallthrough
		case z.BareOutput.Writer == origConfig.BareOutput.Writer:
			// keep the original settings if writer wasn't changed
			z.BareOutput = origConfig.BareOutput
			sameBareOutput = true
		}

		if origConfig.BuildConfig == z.Config.BuildConfig && sameBareOutput {
			// config and output are identical - we can reuse the original logger
			// but we must check for exceptions

			switch { // shall not reuse the original logger if ...
			case needsLowLatency && !origConfig.LoggerOutput.IsLowLatencySupported():
				// ... LL support is missing
			default:
				params := logcommon.CopyLoggerParams{reqs, z.level, z.fields, z.dynFields}
				if logger := template.CopyTemplateLogger(params); logger != nil {
					return logger, nil
				}
			}
			break
		}
		if lo, ok := z.BareOutput.Writer.(logcommon.LoggerOutput); ok {
			// something strange, but we can also work this way
			output = lo
			break
		}
		if sameBareOutput &&
			origConfig.Output.CanReuseOutputFor(z.Output) &&
			origConfig.Instruments.CanReuseOutputFor(z.Instruments) {

			// same output, and it can be reused with the new settings
			output = origConfig.LoggerOutput
			break
		}
	}
	if output == nil || needsLowLatency && !output.IsLowLatencySupported() {
		var err error
		output, err = z.prepareOutput(metrics, needsLowLatency)
		if err != nil {
			return nil, err
		}
	}

	z.Config.Metrics = metrics
	z.Config.LoggerOutput = output

	params := logcommon.NewLoggerParams{reqs, z.level, z.fields, z.dynFields, z.Config}
	return z.factory.CreateNewLogger(params)
}

func (z LoggerBuilder) prepareOutput(metrics *logcommon.MetricsHelper, needsLowLatency bool) (logcommon.LoggerOutput, error) {

	outputWriter, err := z.factory.PrepareBareOutput(z.BareOutput, metrics, z.Config.BuildConfig)
	if err != nil {
		return nil, err
	}

	outputAdapter := logwriter.NewAdapter(outputWriter, z.BareOutput.ProtectedClose,
		z.BareOutput.FlushFn, z.BareOutput.FlushFn)

	if z.Config.Output.ParallelWriters < 0 || z.Config.Output.ParallelWriters > math.MaxUint8 {
		return nil, errors.New("argument ParallelWriters is out of bounds")
	}

	if z.Config.Output.BufferSize > 0 {
		if z.Config.Output.ParallelWriters > 0 && z.Config.Output.ParallelWriters*2 < z.Config.Output.BufferSize {
			// to limit write parallelism - buffer must be active
			return nil, errors.New("write parallelism limiter requires BufferSize >= ParallelWriters*2 ")
		}

		flags := bpbuffer.BufferWriteDelayFairness | bpbuffer.BufferTrackWriteDuration

		if z.Config.Output.BufferSize > 1000 {
			flags |= bpbuffer.BufferDropOnFatal
		}

		if z.factory.CanReuseMsgBuffer() {
			flags |= bpbuffer.BufferReuse
		}

		missedFn := z.loggerMissedEvent(WarnLevel, metrics)

		var bpb *bpbuffer.BackpressureBuffer
		switch {
		case z.Config.Output.EnableRegularBuffer:
			pw := uint8(DefaultOutputParallelLimit)
			if z.Config.Output.ParallelWriters != 0 {
				pw = uint8(z.Config.Output.ParallelWriters)
			}
			bpb = bpbuffer.NewBackpressureBuffer(outputAdapter, z.Config.Output.BufferSize, pw, flags, missedFn)
		case z.Config.Output.ParallelWriters == 0 || z.Config.Output.ParallelWriters == math.MaxUint8:
			bpb = bpbuffer.NewBackpressureBufferWithBypass(outputAdapter, z.Config.Output.BufferSize,
				0 /* no limit */, flags, missedFn)
		default:
			bpb = bpbuffer.NewBackpressureBufferWithBypass(outputAdapter, z.Config.Output.BufferSize,
				uint8(z.Config.Output.ParallelWriters), flags, missedFn)
		}

		bpb.StartWorker(context.Background())
		return bpb, nil
	}

	if needsLowLatency {
		return nil, errors.New("low latency buffer was disabled but is required")
	}

	fdw := logwriter.NewDirectWriter(outputAdapter)
	return fdw, nil
}

func (z LoggerBuilder) loggerMissedEvent(level Level, metrics *logcommon.MetricsHelper) bpbuffer.MissedEventFunc {
	return func(missed int) (Level, []byte) {
		metrics.OnWriteSkip(missed)
		return level, ([]byte)(
			fmt.Sprintf(`{"level":"%v","message":"logger dropped %d messages"}`, level.String(), missed))
	}
}
