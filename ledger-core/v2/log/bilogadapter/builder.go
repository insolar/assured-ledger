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

package bilogadapter

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logmetrics"
)

var binLogGlobalAdapter logcommon.GlobalLogAdapter = binLogGlobal{}

type binLogGlobal struct{}

func (v binLogGlobal) SetGlobalLoggerFilter(level logcommon.LogLevel) {
	setGlobalFilter(level)
}

func (v binLogGlobal) GetGlobalLoggerFilter() logcommon.LogLevel {
	return getGlobalFilter()
}

var _ logadapter.Factory = binLogFactory{}
var _ logcommon.GlobalLogAdapterFactory = binLogFactory{}

type binLogFactory struct{}

func (binLogFactory) GetGlobalLogAdapter() logcommon.GlobalLogAdapter {
	return binLogGlobalAdapter
}

func (binLogFactory) CanReuseMsgBuffer() bool {
	return true
}

func (b binLogFactory) PrepareBareOutput(output logadapter.BareOutput, metrics *logmetrics.MetricsHelper, config logadapter.BuildConfig) (io.Writer, error) {
	//outputWriter, err := selectFormatter(config.Output.Format, output.Writer)
	//
	//if err != nil {
	//	return nil, err
	//}
	//
	//if ok, name, reportFn := getWriteDelayConfig(metrics, config); ok {
	//	outputWriter = newWriteDelayPostHook(outputWriter, name, zf.writeDelayPreferTrim, reportFn)
	//}
	//
	//return outputWriter, nil
	panic("implement me")
}

func (binLogFactory) CreateNewLogger(params logadapter.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	panic("implement me")
}

type binLogTemplate struct {
	binLogFactory
	template *binLogAdapter
}

func (b binLogTemplate) GetLoggerOutput() logcommon.LoggerOutput {
	return b.template.GetLoggerOutput()
}

func (b binLogTemplate) GetTemplateConfig() logadapter.Config {
	return *b.template.config
}

func (b binLogTemplate) CreateNewLogger(params logadapter.NewLoggerParams) (logcommon.EmbeddedLogger, error) {
	panic("implement me")
}

func (b binLogTemplate) CopyTemplateLogger(logadapter.CopyLoggerParams) logcommon.EmbeddedLogger {
	panic("implement me")
}
