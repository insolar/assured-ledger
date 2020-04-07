// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logcommon

import (
	"fmt"
	"strings"
)

type Level uint8

// NoLevel means it should be ignored
const (
	Disabled Level = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	PanicLevel
	FatalLevel
	NoLevel

	LogLevelCount = iota
)
const MinLevel = DebugLevel

func (l Level) IsValid() bool {
	return l > Disabled && l < NoLevel
}

func (l Level) Equal(other Level) bool {
	return l == other
}

func (l Level) String() string {
	switch l {
	case NoLevel:
		return ""
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	case Disabled:
		//
	}
	return "ignore"
}

func ParseLevel(levelStr string) (Level, error) {
	switch strings.ToLower(levelStr) {
	case Disabled.String():
		return Disabled, nil
	case DebugLevel.String():
		return DebugLevel, nil
	case InfoLevel.String():
		return InfoLevel, nil
	case WarnLevel.String():
		return WarnLevel, nil
	case ErrorLevel.String():
		return ErrorLevel, nil
	case FatalLevel.String():
		return FatalLevel, nil
	case PanicLevel.String():
		return PanicLevel, nil
	}
	return Disabled, fmt.Errorf("unknown Level String: '%s', defaulting to NoLevel", levelStr)
}
