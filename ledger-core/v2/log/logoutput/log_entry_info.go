// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logoutput

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	MessageFieldName       = "message"
	TimestampFieldName     = "time"
	LevelFieldName         = "level"
	CallerFieldName        = "caller"
	FuncFieldName          = "func"
	WriteDurationFieldName = "writeDuration"
	StackTraceFieldName    = "errorStack"
	ErrorMsgFieldName      = "errorMsg"
	LogFatalExitCode       = 1
)

func GetCallerInfo(skipCallNumber int) (fileName string, funcName string, line int) {
	return getCallerInfo2(skipCallNumber + 1) // getCallerInfo2 is 1.5 times faster than getCallerInfo1
}

// nolint:unused,deadcode
func getCallerInfo1(skipCallNumber int) (fileName string, funcName string, line int) {
	pc, fileName, line, _ := runtime.Caller(skipCallNumber + 1)

	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	funcName = parts[pl-1]

	if pl > 1 && strings.HasPrefix(parts[pl-2], "(") {
		funcName = parts[pl-2] + "." + funcName
	}

	return fileName, funcName, line
}

func getCallerInfo2(skipCallNumber int) (fileName string, funcName string, line int) {
	pc := make([]uintptr, 1)

	if runtime.Callers(skipCallNumber+2, pc) == 1 {
		pc := pc[0] - 1
		if fn := runtime.FuncForPC(pc); fn != nil {

			parts := strings.Split(fn.Name(), ".")
			pl := len(parts)
			funcName = parts[pl-1]

			if pl > 1 && strings.HasPrefix(parts[pl-2], "(") {
				funcName = parts[pl-2] + "." + funcName
			}

			fileName, line = fn.FileLine(pc)
			return fileName, funcName, line
		}
	}
	return "<unknown>", "<unknown>", 0
}

func GetCallerFileNameWithLine(skipCallNumber, adjustLineNumber int) (fileNameAndLine string, funcName string) {
	fileName, fName, fileLine := GetCallerInfo(skipCallNumber + 1)
	_, fileName = filepath.Split(fileName)
	return fmt.Sprintf("%s:%d", fileName, fileLine+adjustLineNumber), fName
}
