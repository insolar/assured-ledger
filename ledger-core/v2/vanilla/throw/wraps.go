//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package throw

import (
	"errors"
)

type bypassWrapper interface {
	logStringer
	bypassWrapper()
}

type msgWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	msg        string
}

func (v msgWrap) bypassWrapper() {}

func (v msgWrap) Reason() error {
	return v
}

func (v msgWrap) StackTrace() StackTrace {
	return v.st
}

func (v msgWrap) LogString() string {
	return v.msg
}

func (v msgWrap) Error() string {
	if v.st == nil {
		return v.msg
	}
	return v.msg + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}

/*******************************************************************/

type stackWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	err        error
}

func (v stackWrap) bypassWrapper() {}

func (v stackWrap) StackTrace() StackTrace {
	return v.st
}

func (v stackWrap) Reason() error {
	return v.Unwrap()
}

func (v stackWrap) Unwrap() error {
	return v.err
}

func (v stackWrap) LogString() string {
	if vv, ok := v.err.(logStringer); ok {
		return vv.LogString()
	}
	return v.err.Error()
}

func (v stackWrap) Error() string {
	if v.st == nil {
		return v.LogString()
	}
	return v.LogString() + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}

/*******************************************************************/

type panicWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	recovered  interface{}
	fmtWrap
}

func (v fmtWrap) bypassWrapper() {}

func (v panicWrap) Reason() error {
	if err := v.Unwrap(); err != nil {
		return err
	}
	return errors.New(v.LogString())
}

func (v panicWrap) StackTrace() StackTrace {
	return v.st
}

func (v panicWrap) Recovered() interface{} {
	if v.recovered != nil {
		return v.recovered
	}
	return v.fmtWrap
}

func (v panicWrap) Unwrap() error {
	if err, ok := v.recovered.(error); ok {
		return err
	}
	return nil
}

func (v panicWrap) Error() string {
	if v.st == nil {
		return v.LogString()
	}
	return v.LogString() + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}

/*******************************************************************/

type fmtWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	msg        string
	extra      interface{}
}

func (v fmtWrap) extraString() string {
	if vv, ok := v.extra.(logStringer); ok {
		return vv.LogString()
	}
	return ""
}

func (v fmtWrap) LogString() string {
	switch s := v.extraString(); {
	case s == "":
		return v.msg
	case v.msg == "":
		return s
	default:
		return v.msg + "\t" + s
	}
}

func (v fmtWrap) Error() string {
	return v.LogString()
}

func (v fmtWrap) ExtraInfo() interface{} {
	return v.extra
}

/*******************************************************************/

type detailsWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	err        error
	details    fmtWrap
}

func (v detailsWrap) Unwrap() error {
	return v.err
}

func (v detailsWrap) LogString() string {
	s := ""
	if vv, ok := v.err.(logStringer); ok {
		s = vv.LogString()
	} else {
		s = v.err.Error()
	}

	switch m := v.details.LogString(); {
	case s == "":
		return m
	case m == "":
		return s
	default:
		return m + "\t" + s
	}
}

func (v detailsWrap) Is(target error) bool {
	value := v.details.extra
	if x, ok := value.(iser); ok && x.Is(target) {
		return true
	}
	if e, ok := value.(error); ok {
		if x, ok := target.(iser); ok && x.Is(e) {
			return true
		}
	}
	return false
}

func (v detailsWrap) As(target interface{}) bool {
	if x, ok := v.details.extra.(interface{ As(interface{}) bool }); ok && x.As(target) {
		return true
	}
	return false
}

func (v detailsWrap) Error() string {
	return v.LogString()
}

func (v detailsWrap) ExtraInfo() interface{} {
	return v.details.ExtraInfo()
}
