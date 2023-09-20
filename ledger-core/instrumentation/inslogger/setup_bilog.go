package inslogger

import (
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/log/adapters/bilog"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
	"github.com/insolar/assured-ledger/ledger-core/log/logoutput"
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
