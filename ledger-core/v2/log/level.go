// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package log

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logfmt"
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
