//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

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
