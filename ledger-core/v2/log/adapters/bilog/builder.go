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

package bilog

import (
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/bilogencoder"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/json"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog/text"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

/* =========================== */

func NewBuilder(cfg logcommon.Config, level log.Level, encoders bilogencoder.EncoderFactoryDispatcherFunc) log.LoggerBuilder {
	return log.NewBuilder(binLogFactory{encoders}, cfg, level)
}

var _ logcommon.Factory = binLogFactory{}

type binLogFactory struct {
	encoders bilogencoder.EncoderFactoryDispatcherFunc
	//	writeDelayPreferTrim bool
}

func (binLogFactory) GetGlobalLogAdapter() logcommon.GlobalLogAdapter {
	return binLogGlobalAdapter
}

func (binLogFactory) CanReuseMsgBuffer() bool {
	return true
}

func (b binLogFactory) PrepareBareOutput(output logcommon.BareOutput, _ *logcommon.MetricsHelper, config logcommon.BuildConfig) (io.Writer, error) {
	return output.Writer, nil
}

func (b binLogFactory) CreateNewLogger(params logcommon.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	return b.createLogger(params, nil)
}

const minEventBuffer = 512
const maxEventBufferIncrement = minEventBuffer << 1
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

func (b binLogFactory) createLogger(params logcommon.NewLoggerParams, template *binLogAdapter) (logcommon.EmbeddedLogger, error) {

	//if params.Config.Instruments.MetricsMode&logcommon.LogMetricsWriteDelayFlags != 0 {
	//	return nil, errors.New("WriteDelay metric is not supported")
	//}

	var encoderFactory bilogencoder.EncoderFactory

	outFormat := params.Config.Output.Format
	switch outFormat {
	case logcommon.JSONFormat:
		encoderFactory = json.EncoderManager()
	case logcommon.TextFormat:
		encoderFactory = text.EncoderManager()
	default:
		if b.encoders != nil {
			encoderFactory = b.encoders(string(outFormat))
			if encoderFactory != nil {
				break
			}
		}
		return nil, errors.New("unknown output format: " + outFormat.String())
	}
	encoder := encoderFactory.CreateEncoder(params.Config.MsgFormat)
	if encoder == nil {
		return nil, errors.New("encoder factory has failed: " + outFormat.String())
	}

	cfg := params.Config

	la := binLogAdapter{
		config:      &cfg,
		encoder:     encoder,
		writer:      params.Config.LoggerOutput,
		levelFilter: params.Level,
		lowLatency:  params.Reqs&logcommon.RequiresLowLatency != 0,
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
