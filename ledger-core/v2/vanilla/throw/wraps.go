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
	return joinStack(v.msg, v.st)
}

func joinOpt(s0, s1 string) string {
	switch {
	case s0 == "":
		return s1
	case s1 == "":
		return s0
	default:
		return s0 + "\t" + s1
	}
}

func joinStack(s0 string, s1 StackTrace) string {
	if s1 == nil {
		return s0
	}
	return s0 + "\t" + StackTracePrefix + s1.StackTraceAsText()
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
	return joinStack(v.LogString(), v.st)
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
	return joinStack(v.LogString(), v.st)
}

/*******************************************************************/

type fmtWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	msg        string
	extra      interface{}
	useExtra   bool
}

func (v fmtWrap) extraString() string {
	if !v.useExtra {
		return ""
	}
	if vv, ok := v.extra.(logStringer); ok {
		return vv.LogString()
	}
	return defaultFmt(v.extra, false)
}

func (v fmtWrap) LogString() string {
	return joinOpt(v.msg, v.extraString())
}

func (v fmtWrap) Error() string {
	return v.LogString()
}

func (v fmtWrap) ExtraInfo() interface{} {
	return v.extra
}

/*******************************************************************/

type detailsWrap struct {
	_logignore   struct{} // will be ignored by struct-logger
	err          error
	details      fmtWrap
	isComparable bool
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

	return joinOpt(v.details.LogString(), s)
}

func (v detailsWrap) Is(target error) bool {
	if e, ok := v.details.extra.(error); ok {
		return isThis(v.isComparable, e, target)
	}
	return false
}

func (v detailsWrap) As(target interface{}) bool {
	if e, ok := v.details.extra.(error); ok {
		fnAs := errors.As // to avoid GoLang warning on use of errors.As
		return fnAs(e, target)
	}
	return false
}

func (v detailsWrap) Error() string {
	return v.LogString()
}

func (v detailsWrap) ExtraInfo() interface{} {
	return v.details.ExtraInfo()
}
