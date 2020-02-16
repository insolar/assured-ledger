// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"errors"
)

type bypassWrapper interface {
	logStringer
	bypassWrapper()
}

type msgWrap struct {
	st  StackTrace
	msg string
}

func (v msgWrap) bypassWrapper() {}

func (v msgWrap) Cause() error {
	return v
}

func (v msgWrap) ShallowStackTrace() StackTrace {
	return v.st
}

func (v msgWrap) DeepestStackTrace() StackTrace {
	return v.st
}

func (v msgWrap) LogString() string {
	return v.msg
}

func (v msgWrap) Error() string {
	return joinStack(v.msg, v.st)
}

func joinErrString(s0, s1 string) string {
	switch {
	case s0 == "":
		return s1
	case s1 == "":
		return s0
	default:
		return s0 + ";\t" + s1
	}
}

func joinStack(s0 string, s1 StackTrace) string {
	if s1 == nil {
		return s0
	}
	return s0 + "\n" + stackTracePrintPrefix + s1.StackTraceAsText()
}

/*******************************************************************/

type stackWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	stDeepest  StackTrace
	err        error
}

func (v stackWrap) bypassWrapper() {}

func (v stackWrap) ShallowStackTrace() StackTrace {
	return v.st
}

func (v stackWrap) DeepestStackTrace() StackTrace {
	if v.stDeepest == nil {
		return v.st
	}
	return v.stDeepest
}

func (v stackWrap) Cause() error {
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
	if v.st == nil || v.stDeepest != nil {
		return v.LogString()
	}
	return joinStack(v.LogString(), v.st)
}

/*******************************************************************/

type panicWrap struct {
	st        StackTrace
	stDeepest StackTrace
	recovered interface{}
	fmtWrap
}

func (v fmtWrap) bypassWrapper() {}

func (v panicWrap) Cause() error {
	if err := v.Unwrap(); err != nil {
		return err
	}
	return errors.New(v.LogString())
}

func (v panicWrap) ShallowStackTrace() StackTrace {
	return v.st
}

func (v panicWrap) DeepestStackTrace() StackTrace {
	if v.stDeepest == nil {
		return v.st
	}
	return v.stDeepest
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
	return joinStack(v.LogString(), v.DeepestStackTrace())
}

/*******************************************************************/

type fmtWrap struct {
	msg      string
	extra    interface{}
	useExtra bool // indicates that extra part is included into msg
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
	return joinErrString(v.msg, v.extraString())
}

func (v fmtWrap) Error() string {
	return v.LogString()
}

func (v fmtWrap) ExtraInfo() (string, interface{}) {
	if !v.useExtra {
		return "", v.extra
	}
	return v.msg, v.extra
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

	return joinErrString(v.details.LogString(), s)
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
	return joinErrString(v.details.LogString(), v.err.Error())
}

func (v detailsWrap) ExtraInfo() (string, interface{}) {
	return v.details.ExtraInfo()
}
