// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

// NewBuilderWithTemplate returns new LoggerBuilder from given template
func NewBuilderWithTemplate(template logcommon.Template, level Level) LoggerBuilder {
	config := template.GetTemplateConfig()
	return LoggerBuilder{
		factory:     template,
		hasTemplate: true,
		level:       level,
		cfg:         config,
	}
}

// NewBuilder returns new LoggerBuilder with given factory
func NewBuilder(factory logcommon.Factory, config logcommon.Config, level Level) LoggerBuilder {
	return LoggerBuilder{
		factory: factory,
		level:   level,
		cfg:     config,
	}
}

// LoggerBuilder is used to build new Logger with extra persistent and dynamic fields.
// It ensures layer of abstraction between logger itself, extra fields storage and writer
type LoggerBuilder struct {
	factory     logcommon.Factory
	hasTemplate bool

	level Level

	noFields    bool
	noDynFields bool

	fields    map[string]interface{}
	dynFields logcommon.DynFieldMap

	cfg logcommon.Config
}

// IsZero returns true if logger has valid factory
func (z LoggerBuilder) IsZero() bool {
	return z.factory == nil
}

// GetOutput returns the current output destination/writer
func (z LoggerBuilder) GetOutput() io.Writer {
	return z.cfg.BareOutput.Writer
}

// GetLogLevel returns the current log level
func (z LoggerBuilder) GetLogLevel() Level {
	return z.level
}

// WithOutput sets the output destination/writer for the logger.
// Argument of LoggerOutput type will allow fine-grain control of events, but it may ignore WithFormat (depends on adapter).
func (z LoggerBuilder) WithOutput(w io.Writer) LoggerBuilder {

	z.cfg.BareOutput = logcommon.BareOutput{Writer: w}
	switch ww := w.(type) {
	case interface{ Flush() error }:
		z.cfg.BareOutput.FlushFn = ww.Flush
	case interface{ Sync() error }:
		z.cfg.BareOutput.FlushFn = ww.Sync
	}

	return z
}

// WithBuffer sets buffer size and applicability of the buffer. Will be IGNORED when a reused output is already buffered.
func (z LoggerBuilder) WithBuffer(bufferSize int, bufferForAll bool) LoggerBuilder {
	z.cfg.Output.BufferSize = bufferSize
	z.cfg.Output.EnableRegularBuffer = bufferForAll
	return z
}

// WithLevel sets log level.
func (z LoggerBuilder) WithLevel(level Level) LoggerBuilder {
	z.level = level
	return z
}

// WithFormat sets logger output format.
// Format support depends on the log adapter. Unsupported format will fail on Build().
// NB! Changing log format may clear out inherited non-dynamic fields (depends on adapter).
// NB! Format can be ignored when WithOutput(LoggerOutput) is applied (depends on adapter).
func (z LoggerBuilder) WithFormat(format logcommon.LogFormat) LoggerBuilder {
	z.cfg.Output.Format = format
	return z
}

// Controls 'func' and 'caller' field computation. See also WithSkipFrameCount().
func (z LoggerBuilder) WithCaller(mode logcommon.CallerFieldMode) LoggerBuilder {
	z.cfg.Instruments.CallerMode = mode
	return z
}

// WithSkipFrameCount allows customization of skip frames for 'func' and 'caller' fields.
func (z LoggerBuilder) WithSkipFrameCount(skipFrameCount int) LoggerBuilder {
	if skipFrameCount < math.MinInt8 || skipFrameCount > math.MaxInt8 {
		panic("illegal value")
	}
	z.cfg.Instruments.SkipFrameCount = int8(skipFrameCount)
	return z
}

// Controls collection of metrics. Required flags are ADDED to the current flags. Include specify LogMetricsResetMode to replace flags.
func (z LoggerBuilder) WithMetrics(mode logcommon.LogMetricsMode) LoggerBuilder {
	if mode&logcommon.LogMetricsResetMode != 0 {
		z.cfg.Instruments.MetricsMode = 0
		mode &^= logcommon.LogMetricsResetMode
	}
	z.cfg.Instruments.MetricsMode |= mode
	return z
}

// WithMetricsRecorder sets an custom recorder for metric collection.
func (z LoggerBuilder) WithMetricsRecorder(recorder logcommon.LogMetricsRecorder) LoggerBuilder {
	z.cfg.Instruments.Recorder = recorder
	return z
}

// WithoutInheritedFields clears out inherited fields (dynamic or not)
func (z LoggerBuilder) WithoutInheritedFields() LoggerBuilder {
	z.noFields = true
	z.noDynFields = true
	return z
}

// WithoutInheritedDynFields clears out inherited dynamic fields only
func (z LoggerBuilder) WithoutInheritedDynFields() LoggerBuilder {
	z.noDynFields = true
	return z
}

// WithFields adds fields for to-be-built logger. Fields are deduplicated within a single builder only.
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

// WithField add a fields for to-be-built logger. Fields are deduplicated within a single builder only.
func (z LoggerBuilder) WithField(k string, v interface{}) LoggerBuilder {
	if z.fields == nil {
		z.fields = make(map[string]interface{})
	}
	delete(z.dynFields, k)
	z.fields[k] = v
	return z
}

// Adds a dynamically-evaluated field. Fields are deduplicated. When func=nil or func()=nil then the field is omitted.
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

// MustBuild creates a logger. Panics on error.
func (z LoggerBuilder) MustBuild() Logger {
	if l, err := z.build(false); err != nil {
		panic(err)
	} else {
		return l
	}
}

// Build creates a logger.
func (z LoggerBuilder) Build() (Logger, error) {
	return z.build(false)
}

// BuildLowLatency creates a logger with no write delays.
func (z LoggerBuilder) BuildLowLatency() (Logger, error) {
	return z.build(true)
}

func (z LoggerBuilder) build(needsLowLatency bool) (Logger, error) {
	el, err := z.buildEmbedded(needsLowLatency)
	if err != nil {
		return Logger{}, err
	}
	return WrapEmbeddedLogger(el), nil
}

func (z LoggerBuilder) buildEmbedded(needsLowLatency bool) (logcommon.EmbeddedLogger, error) {
	var metrics *logcommon.MetricsHelper

	if z.cfg.Instruments.MetricsMode != logcommon.NoLogMetrics {
		metrics = logcommon.NewMetricsHelper(z.cfg.Instruments.Recorder)
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
	case z.cfg.BareOutput.Writer == nil:
		return nil, errors.New("output is nil")
	case z.hasTemplate:
		template := z.factory.(logcommon.Template)
		origConfig := template.GetTemplateConfig()

		sameBareOutput := false
		switch {
		case z.cfg.BareOutput.Writer == origConfig.LoggerOutput: // users can be crazy
			fallthrough
		case z.cfg.BareOutput.Writer == origConfig.BareOutput.Writer:
			// keep the original settings if writer wasn't changed
			z.cfg.BareOutput = origConfig.BareOutput
			sameBareOutput = true
		}

		if origConfig.BuildConfig == z.cfg.BuildConfig && sameBareOutput {
			// config and output are identical - we can reuse the original logger
			// but we must check for exceptions

			switch { // shall not reuse the original logger if ...
			case needsLowLatency && !origConfig.LoggerOutput.IsLowLatencySupported():
				// ... LL support is missing
			default:
				params := logcommon.CopyLoggerParams{
					Reqs: reqs, Level: z.level, AppendFields: z.fields, AppendDynFields: z.dynFields}
				if logger := template.CopyTemplateLogger(params); logger != nil {
					return logger, nil
				}
			}
			break
		}
		if lo, ok := z.cfg.BareOutput.Writer.(logcommon.LoggerOutput); ok {
			// something strange, but we can also work this way
			output = lo
			break
		}
		if sameBareOutput &&
			origConfig.Output.CanReuseOutputFor(z.cfg.Output) &&
			origConfig.Instruments.CanReuseOutputFor(z.cfg.Instruments) {

			// same output, and it can be reused with the new settings
			output = origConfig.LoggerOutput
			break
		}
	default:
		if lo, ok := z.cfg.BareOutput.Writer.(logcommon.LoggerOutput); ok {
			// something strange, but we can also work this way
			output = lo
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

	z.cfg.Metrics = metrics
	z.cfg.LoggerOutput = output

	params := logcommon.NewLoggerParams{
		Reqs: reqs, Level: z.level, Fields: z.fields, DynFields: z.dynFields, Config: z.cfg}
	return z.factory.CreateNewLogger(params)
}

func (z LoggerBuilder) prepareOutput(metrics *logcommon.MetricsHelper, needsLowLatency bool) (logcommon.LoggerOutput, error) {

	outputWriter, err := z.factory.PrepareBareOutput(z.cfg.BareOutput, metrics, z.cfg.BuildConfig)
	if err != nil {
		return nil, err
	}
	//if _, ok := z.cfg.BareOutput.(logcommon.LoggerOutput); ok &&

	outputAdapter := logwriter.NewAdapter(outputWriter, z.cfg.BareOutput.ProtectedClose,
		z.cfg.BareOutput.FlushFn, z.cfg.BareOutput.FlushFn)

	if z.cfg.Output.ParallelWriters < 0 || z.cfg.Output.ParallelWriters > math.MaxUint8 {
		return nil, errors.New("argument ParallelWriters is out of bounds")
	}

	if z.cfg.Output.BufferSize > 0 {
		if z.cfg.Output.ParallelWriters > 0 && z.cfg.Output.ParallelWriters*2 < z.cfg.Output.BufferSize {
			// to limit write parallelism - buffer must be active
			return nil, errors.New("write parallelism limiter requires BufferSize >= ParallelWriters*2 ")
		}

		flags := bpbuffer.BufferWriteDelayFairness | bpbuffer.BufferTrackWriteDuration

		if z.cfg.Output.BufferSize > 1000 {
			flags |= bpbuffer.BufferDropOnFatal
		}

		if z.factory.CanReuseMsgBuffer() {
			flags |= bpbuffer.BufferReuse
		}

		missedFn := z.loggerMissedEvent(WarnLevel, metrics)

		var bpb *bpbuffer.BackpressureBuffer
		switch {
		case z.cfg.Output.EnableRegularBuffer:
			pw := uint8(DefaultOutputParallelLimit)
			if z.cfg.Output.ParallelWriters != 0 {
				pw = uint8(z.cfg.Output.ParallelWriters)
			}
			bpb = bpbuffer.NewBackpressureBuffer(outputAdapter, z.cfg.Output.BufferSize, pw, flags, missedFn)
		case z.cfg.Output.ParallelWriters == 0 || z.cfg.Output.ParallelWriters == math.MaxUint8:
			bpb = bpbuffer.NewBackpressureBufferWithBypass(outputAdapter, z.cfg.Output.BufferSize,
				0 /* no limit */, flags, missedFn)
		default:
			bpb = bpbuffer.NewBackpressureBufferWithBypass(outputAdapter, z.cfg.Output.BufferSize,
				uint8(z.cfg.Output.ParallelWriters), flags, missedFn)
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
