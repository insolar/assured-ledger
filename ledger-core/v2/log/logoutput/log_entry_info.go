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
	LogFatalExitCode       = 1
)

func GetCallerInfo(skipCallNumber int) (fileName string, funcName string, line int) {
	pc, fileName, line, _ := runtime.Caller(skipCallNumber + 1)

	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	funcName = parts[pl-1]

	if pl > 1 && strings.HasPrefix(parts[pl-2], "(") {
		funcName = parts[pl-2] + "." + funcName
	}

	return fileName, funcName, line
}

func GetCallerFileNameWithLine(skipCallNumber, adjustLineNumber int) (fileNameLine string, funcName string) {
	fileName, funcName, fileLine := GetCallerInfo(skipCallNumber + 1)
	_, fileName = filepath.Split(fileName)
	return fmt.Sprintf("%s:%d", fileName, fileLine+adjustLineNumber), funcName
}
