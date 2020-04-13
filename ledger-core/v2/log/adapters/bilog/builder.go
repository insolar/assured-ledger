// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bilog

import (
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/pbuf"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/json"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/msgencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/text"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

// TODO PLAT-45 test performance and memory impact of (recycleBuf)

func NewFactory(encoders msgencoder.FactoryDispatcherFunc, recycleBuf bool) logcommon.Factory {
	return binLogFactory{encoders, recycleBuf}
}

var _ logcommon.Factory = binLogFactory{}

type binLogFactory struct {
	encoders   msgencoder.FactoryDispatcherFunc
	recycleBuf bool
	//	writeDelayPreferTrim bool
}

func (binLogFactory) GetGlobalLogAdapter() logcommon.GlobalLogAdapter {
	return binLogGlobalAdapter
}

func (f binLogFactory) CanReuseMsgBuffer() bool {
	return !f.recycleBuf
}

func (f binLogFactory) PrepareBareOutput(output logcommon.BareOutput, metrics *logcommon.MetricsHelper, config logcommon.BuildConfig) (io.Writer, error) {
	if !config.Instruments.MetricsMode.HasWriteMetric() {
		return output.Writer, nil
	}

	encoder, err := f.createEncoderFactory(config.Output.Format)
	if err != nil {
		return nil, err
	}

	var reportFn logcommon.DurationReportFunc
	if config.Instruments.MetricsMode&logcommon.LogMetricsWriteDelayReport != 0 {
		reportFn = metrics.GetOnWriteDurationReport()
	}

	if config.Instruments.MetricsMode&logcommon.LogMetricsWriteDelayField != 0 {
		return encoder.CreateMetricWriter(output.Writer, logoutput.WriteDurationFieldName, reportFn)
	}
	return encoder.CreateMetricWriter(output.Writer, "", reportFn)
}

func (f binLogFactory) CreateNewLogger(params logcommon.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	return f.createLogger(params, nil)
}

func (f binLogFactory) createEncoderFactory(outFormat logcommon.LogFormat) (msgencoder.EncoderFactory, error) {
	switch outFormat {
	case logcommon.JSONFormat:
		return json.EncoderManager(), nil
	case logcommon.TextFormat:
		return text.EncoderManager(), nil
	case logcommon.PbufFormat:
		return pbuf.EncoderManager(), nil
	default:
		if f.encoders == nil {
			break
		}
		if encoderFactory := f.encoders(string(outFormat)); encoderFactory != nil {
			return encoderFactory, nil
		}
	}
	return nil, errors.New("unknown output format: " + outFormat.String())
}

const minEventBuffer = 512
const maxEventBufferIncrement = minEventBuffer >> 1
const anticipatedFieldBuffer = 32

func estimateBufferSizeByFields(nCount int) int {
	if nCount == 0 {
		return 0
	}
	if sz := nCount * anticipatedFieldBuffer; sz < maxEventBufferIncrement {
		return sz
	}
	return maxEventBufferIncrement
}

func (f binLogFactory) createLogger(params logcommon.NewLoggerParams, template *binLogAdapter) (logcommon.EmbeddedLogger, error) {

	outFormat := params.Config.Output.Format

	if template != nil && outFormat != template.config.Output.Format {
		// no field inheritance when format is changed
		params.Reqs &^= logcommon.RequiresParentCtxFields
	}

	// ========== DO NOT modify config beyond this point ================
	cfg := params.Config // ensure a separate copy on heap

	la := binLogAdapter{
		config:      &cfg,
		writer:      params.Config.LoggerOutput,
		levelFilter: params.Level,
		recycleBuf:  f.recycleBuf,
		lowLatency:  params.Reqs&logcommon.RequiresLowLatency != 0,

		expectedEventLen: anticipatedFieldBuffer,
	}

	encoderFactory, err := f.createEncoderFactory(outFormat)
	if err != nil {
		return nil, err
	}
	la.encoder = encoderFactory.CreateEncoder(params.Config.MsgFormat)
	if la.encoder == nil {
		return nil, errors.New("encoder factory has failed: " + outFormat.String())
	}

	{ // replacement and inheritance for ctxFields
		switch {
		case template == nil || params.Reqs&logcommon.RequiresParentCtxFields == 0:
			//
		case len(template.parentStatic) == 0:
			la.parentStatic = template.staticFields
		default:
			la.parentStatic = template.parentStatic
			la.staticFields = template.staticFields
		}

		la._addFieldsByBuilder(params.Fields)

		la.expectedEventLen += len(la.parentStatic)
		la.expectedEventLen += len(la.staticFields)
	}

	{ // replacement and inheritance for dynFields
		if template != nil && params.Reqs&logcommon.RequiresParentDynFields != 0 && len(template.dynFields) > 0 {
			la.dynFields = template.dynFields
		}

		la._addDynFieldsByBuilder(params.DynFields)

		la.expectedEventLen += estimateBufferSizeByFields(len(la.dynFields))
	}

	if la.expectedEventLen > maxEventBufferIncrement {
		la.expectedEventLen += maxEventBufferIncrement
	}

	return &la, nil
}

/* =========================== */

type binLogTemplate struct {
	binLogFactory
	template *binLogAdapter
}

func (b binLogTemplate) GetTemplateLevel() logcommon.Level {
	return b.template.levelFilter
}

func (b binLogTemplate) GetLoggerOutput() logcommon.LoggerOutput {
	return b.template.GetLoggerOutput()
}

func (b binLogTemplate) GetTemplateConfig() logcommon.Config {
	return *b.template.config
}

func (b binLogTemplate) CreateNewLogger(params logcommon.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	return b.createLogger(params, b.template)
}

func (b binLogTemplate) CopyTemplateLogger(params logcommon.CopyLoggerParams) logcommon.EmbeddedLogger {

	hasUpdates := false
	la := *b.template
	la.expectedEventLen = 0

	if la.levelFilter != params.Level {
		la.levelFilter = params.Level
		hasUpdates = true
	}

	{ // replacement and inheritance for ctxFields
		switch {
		case params.Reqs&logcommon.RequiresParentCtxFields == 0:
			if la.parentStatic != nil || la.staticFields != nil {
				la.parentStatic = nil
				la.staticFields = nil
				hasUpdates = true
			}
		case len(la.parentStatic) == 0:
			la.parentStatic = la.staticFields
			la.staticFields = nil
		}

		if len(params.AppendFields) > 0 {
			la._addFieldsByBuilder(params.AppendFields)
			hasUpdates = true
		}

		la.expectedEventLen += len(la.parentStatic)
		la.expectedEventLen += len(la.staticFields)
	}

	{ // replacement and inheritance for dynFields
		if params.Reqs&logcommon.RequiresParentDynFields == 0 && la.dynFields != nil {
			la.dynFields = nil
			hasUpdates = true
		}

		if len(params.AppendDynFields) > 0 {
			la._addDynFieldsByBuilder(params.AppendDynFields)
			hasUpdates = true
		}

		la.expectedEventLen += estimateBufferSizeByFields(len(la.dynFields))
	}

	if !hasUpdates {
		return b.template
	}

	if la.expectedEventLen > maxEventBufferIncrement {
		la.expectedEventLen += maxEventBufferIncrement
	}

	return &la
}

/* =========================== */

var binLogGlobalAdapter logcommon.GlobalLogAdapter = binLogGlobal{}

type binLogGlobal struct{}

func (binLogGlobal) SetGlobalLoggerFilter(level log.Level) {
	setGlobalFilter(level)
}

func (binLogGlobal) GetGlobalLoggerFilter() log.Level {
	return getGlobalFilter()
}
