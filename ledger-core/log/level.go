package log

import (
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/log/logfmt"
)

type Level = logcommon.Level

// NoLevel means it should be ignored
const (
	Disabled   = logcommon.Disabled
	DebugLevel = logcommon.DebugLevel
	InfoLevel  = logcommon.InfoLevel
	WarnLevel  = logcommon.WarnLevel
	ErrorLevel = logcommon.ErrorLevel
	FatalLevel = logcommon.FatalLevel
	PanicLevel = logcommon.PanicLevel
	NoLevel    = logcommon.NoLevel

	LevelCount = logcommon.LogLevelCount
)
const MinLevel = logcommon.MinLevel

func ParseLevel(levelStr string) (Level, error) {
	return logcommon.ParseLevel(levelStr)
}

var _ logfmt.LogObject = &Msg{}

type Msg = logfmt.MsgTemplate
