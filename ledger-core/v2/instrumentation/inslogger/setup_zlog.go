// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inslogger

import (
	"errors"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/zlog"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
)

func initZlog() {
	zerolog.CallerMarshalFunc = fileLineMarshaller
	zerolog.TimeFieldFormat = TimestampFormat
	zerolog.CallerFieldName = logoutput.CallerFieldName
	zerolog.LevelFieldName = logoutput.LevelFieldName
	zerolog.TimestampFieldName = logoutput.TimestampFieldName
	zerolog.MessageFieldName = logoutput.MessageFieldName
}

func newZerologAdapter(pCfg ParsedLogConfig, msgFmt logfmt.MsgFormatConfig) (log.LoggerBuilder, error) {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(pCfg.OutputType, pCfg.Output.Format, pCfg.OutputParam)
	if err != nil {
		return log.LoggerBuilder{}, err
	}
	if zc.BareOutput.Writer == nil {
		return log.LoggerBuilder{}, errors.New("output is nil")
	}

	sfb := zlog.ZerologSkipFrameCount + pCfg.SkipFrameBaselineAdjustment
	if sfb < 0 {
		sfb = 0
	}

	zc.Output = pCfg.Output
	zc.Instruments = pCfg.Instruments
	zc.MsgFormat = msgFmt
	zc.MsgFormat.TimeFmt = TimestampFormat // NB! this is ignored by the encoder, it uses zerolog.TimeFieldFormat instead
	zc.Instruments.SkipFrameCountBaseline = uint8(sfb)

	return log.NewBuilder(zlog.NewFactory(), zc, pCfg.LogLevel), nil
}
