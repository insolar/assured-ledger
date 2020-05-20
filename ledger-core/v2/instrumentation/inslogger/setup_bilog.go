// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inslogger

import (
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/adapters/bilog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logoutput"
)

func initBilog() {
	bilog.CallerMarshalFunc = fileLineMarshaller
	//zerolog.TimeFieldFormat = TimestampFormat
}

func newBilogAdapter(pCfg ParsedLogConfig, msgFmt logfmt.MsgFormatConfig) (log.LoggerBuilder, error) {
	zc := logcommon.Config{}

	var err error
	zc.BareOutput, err = logoutput.OpenLogBareOutput(pCfg.OutputType, pCfg.Output.Format, pCfg.OutputParam)
	if err != nil {
		return log.LoggerBuilder{}, err
	}
	if zc.BareOutput.Writer == nil {
		return log.LoggerBuilder{}, errors.New("output is nil")
	}

	sfb := pCfg.SkipFrameBaselineAdjustment
	if sfb < 0 {
		sfb = 0
	}

	zc.Output = pCfg.Output
	zc.Instruments = pCfg.Instruments
	zc.MsgFormat = msgFmt
	zc.MsgFormat.TimeFmt = TimestampFormat
	zc.Instruments.SkipFrameCountBaseline = uint8(sfb)

	return log.NewBuilder(bilog.NewFactory(nil, false), zc, pCfg.LogLevel), nil
}
