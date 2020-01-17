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

package log

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/log/logcommon"
)

type LogLevel = logcommon.Level

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

func ParseLevel(levelStr string) (LogLevel, error) {
	return logcommon.ParseLevel(levelStr)
}
