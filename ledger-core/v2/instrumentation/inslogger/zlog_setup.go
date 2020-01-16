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

package inslogger

import (
	"errors"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logadapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/zlogadapter"
)

func initZlog() {
	zerolog.CallerMarshalFunc = fileLineMarshaller
	zerolog.TimeFieldFormat = TimestampFormat
	zerolog.CallerFieldName = logadapter.CallerFieldName
	zerolog.LevelFieldName = logadapter.LevelFieldName
	zerolog.TimestampFieldName = logadapter.TimestampFieldName
	zerolog.MessageFieldName = logadapter.MessageFieldName
}

func newZerologAdapter(pCfg ParsedLogConfig, msgFmt logadapter.MsgFormatConfig) (logcommon.LoggerBuilder, error) {
	zc := logadapter.Config{}

	var err error
	zc.BareOutput, err = logadapter.OpenLogBareOutput(pCfg.OutputType, pCfg.OutputParam)
	if err != nil {
		return nil, err
	}
	if zc.BareOutput.Writer == nil {
		return nil, errors.New("output is nil")
	}

	sfb := zlogadapter.ZerologSkipFrameCount + pCfg.SkipFrameBaselineAdjustment
	if sfb < 0 {
		sfb = 0
	}

	zc.Output = pCfg.Output
	zc.Instruments = pCfg.Instruments
	zc.MsgFormat = msgFmt
	zc.Instruments.SkipFrameCountBaseline = uint8(sfb)

	return zlogadapter.NewBuilder(zc, pCfg.LogLevel), nil
}
