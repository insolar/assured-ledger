// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package zlog

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

func init() {
	initLevelMappings()
}

type zerologMapping struct {
	zl      zerolog.Level
	fn      func(*zerolog.Logger) *zerolog.Event
	metrics context.Context
}

func (v zerologMapping) IsEmpty() bool {
	return v.fn == nil
}

var zerologLevelMapping = []zerologMapping{
	logcommon.DebugLevel: {zl: zerolog.DebugLevel, fn: (*zerolog.Logger).Debug},
	logcommon.InfoLevel:  {zl: zerolog.InfoLevel, fn: (*zerolog.Logger).Info},
	logcommon.WarnLevel:  {zl: zerolog.WarnLevel, fn: (*zerolog.Logger).Warn},
	logcommon.ErrorLevel: {zl: zerolog.ErrorLevel, fn: (*zerolog.Logger).Error},
	logcommon.FatalLevel: {zl: zerolog.FatalLevel, fn: func(z *zerolog.Logger) *zerolog.Event { return z.WithLevel(zerolog.FatalLevel) }},
	logcommon.PanicLevel: {zl: zerolog.PanicLevel, fn: (*zerolog.Logger).Panic},
	logcommon.NoLevel:    {zl: zerolog.NoLevel, fn: (*zerolog.Logger).Log},
	logcommon.Disabled:   {zl: zerolog.Disabled, fn: func(*zerolog.Logger) *zerolog.Event { return nil }},
}

var zerologReverseMapping []logcommon.Level

func initLevelMappings() {
	var zLevelMax zerolog.Level
	for i := range zerologLevelMapping {
		if zerologLevelMapping[i].IsEmpty() {
			continue
		}
		if zLevelMax < zerologLevelMapping[i].zl {
			zLevelMax = zerologLevelMapping[i].zl
		}
		zerologLevelMapping[i].metrics = logcommon.GetLogLevelContext(logcommon.Level(i))
	}

	zerologReverseMapping = make([]logcommon.Level, zLevelMax+1)
	for i := range zerologReverseMapping {
		zerologReverseMapping[i] = logcommon.Disabled
	}

	for i := range zerologLevelMapping {
		if zerologLevelMapping[i].IsEmpty() {
			zerologLevelMapping[i] = zerologLevelMapping[logcommon.Disabled]
		} else {
			zl := zerologLevelMapping[i].zl
			if zerologReverseMapping[zl] != logcommon.Disabled {
				panic("duplicate level mapping")
			}
			zerologReverseMapping[zl] = logcommon.Level(i)
		}
	}
}

func getLevelMapping(insLevel logcommon.Level) zerologMapping {
	if int(insLevel) > len(zerologLevelMapping) {
		return zerologLevelMapping[logcommon.Disabled]
	}
	return zerologLevelMapping[insLevel]
}

func ToZerologLevel(insLevel logcommon.Level) zerolog.Level {
	return getLevelMapping(insLevel).zl
}

func FromZerologLevel(zLevel zerolog.Level) logcommon.Level {
	if int(zLevel) > len(zerologReverseMapping) {
		return zerologReverseMapping[zerolog.Disabled]
	}
	return zerologReverseMapping[zLevel]
}
