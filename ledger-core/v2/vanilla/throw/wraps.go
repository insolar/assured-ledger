// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"errors"
)

type msgWrap struct {
	st  StackTrace
	msg string
}

func (v msgWrap) Cause() error {
	return errString(v.msg)
}

func (v msgWrap) ShallowStackTrace() StackTrace {
	return v.st
}

func (v msgWrap) DeepestStackTrace() (StackTrace, DeepestStackMode) {
	return v.st, 0
}

func (v msgWrap) ExtraInfo() (string, Severity, interface{}) {
	return v.msg, 0, nil
}

func (v msgWrap) LogString() string {
	return v.msg
}

func (v msgWrap) Error() string {
	return v.msg
}

type errString string

func (v errString) Error() string {
	return string(v)
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

func JoinStackText(s0 string, s1 StackTrace) string {
	if s1 == nil {
		return s0
	}
	return s0 + "\n" + stackTracePrintPrefix + s1.StackTraceAsText()
}

/*******************************************************************/

type stackWrap struct {
	st        StackTrace
	stDeepest StackTrace
	stDeepMod DeepestStackMode
	err       error
}

func (v stackWrap) ShallowStackTrace() StackTrace {
	return v.st
}

func (v stackWrap) DeepestStackTrace() (StackTrace, DeepestStackMode) {
	if v.stDeepest == nil {
		return v.st, v.stDeepMod
	}
	return v.stDeepest, v.stDeepMod
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
	return v.LogString()
}

/*******************************************************************/

type panicWrap struct {
	st        StackTrace
	recovered interface{}
	stDeepest StackTrace
	fmtWrap
	stDeepMod DeepestStackMode
}

func (v panicWrap) Cause() error {
	if err := v.Unwrap(); err != nil {
		return err
	}
	return errors.New(v.LogString())
}

func (v panicWrap) ShallowStackTrace() StackTrace {
	return v.st
}

func (v panicWrap) DeepestStackTrace() (StackTrace, DeepestStackMode) {
	if v.stDeepest == nil {
		return v.st, v.stDeepMod
	}
	return v.stDeepest, v.stDeepMod
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
	return v.LogString()
}

/*******************************************************************/

type fmtWrap struct {
	msg      string
	extra    interface{}
	severity Severity
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

func (v fmtWrap) ExtraInfo() (string, Severity, interface{}) {
	if !v.useExtra {
		return "", v.severity, v.extra
	}
	return v.msg, v.severity, v.extra
}

func (v fmtWrap) AsDetail(target interface{}) bool {
	return asDetail(v.extra, target)
}

/*******************************************************************/

type detailsWrap struct {
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

func (v detailsWrap) AsDetail(target interface{}) bool {
	return v.details.AsDetail(target)
}

func (v detailsWrap) Error() string {
	return joinErrString(v.details.LogString(), v.err.Error())
}

func (v detailsWrap) ExtraInfo() (string, Severity, interface{}) {
	return v.details.ExtraInfo()
}

/*******************************************************************/

type severityWrap struct {
	err      error
	severity Severity
}

func (v severityWrap) Unwrap() error {
	return v.err
}

func (v severityWrap) LogString() string {
	if vv, ok := v.err.(logStringer); ok {
		return vv.LogString()
	}
	return v.err.Error()
}

func (v severityWrap) Error() string {
	return v.err.Error()
}

func (v severityWrap) ExtraInfo() (string, Severity, interface{}) {
	return "", v.severity, nil
}
