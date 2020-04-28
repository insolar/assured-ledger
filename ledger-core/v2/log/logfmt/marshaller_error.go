// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"errors"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type errorMarshaller struct {
	reports  []errorReport
	stack    throw.StackTrace
	msg      string
	severity throw.Severity
	hasPanic bool
}

type errorReport struct {
	marshaller LogObjectMarshaller
	strValue   string
	stack      throw.StackTrace
}

func (v *errorMarshaller) appendReport(m LogObjectMarshaller, s string, ps *string, st throw.StackTrace) bool {
	switch {
	case ps == nil:
		//
	case s == "":
		s = *ps
	case s != "":
		s2 := *ps
		if s2 != "" {
			s += "\t" + s2
		}
	}
	if m != nil || s != "" {
		v.reports = append(v.reports, errorReport{m, s, st})
		return true
	}
	if st != nil {
		v.reports = append(v.reports, errorReport{m, s, st})
	}
	return false
}

func (v *errorMarshaller) appendError(mf MarshallerFactory, x interface{}, st throw.StackTrace) bool {
	m, ps := fmtLogStruct(x, mf, true, true)
	return v.appendReport(m, "", ps, st)
}

func (v *errorMarshaller) appendExtraInfo(mf MarshallerFactory, x interface{}) bool {
	if extras, severity, extra, ok := throw.UnwrapExtraInfo(x); ok {
		if v.severity == 0 && severity != 0 {
			v.severity = severity
		}
		m, ps := fmtLogStruct(extra, mf, false, true)
		if v.appendReport(m, extras, ps, nil) {
			return true
		}
		if err, ok := extra.(error); ok {
			return v.appendReport(nil, err.Error(), nil, nil)
		}
	}
	return false
}

func (v *errorMarshaller) fillLevels(errChain error, mf MarshallerFactory) bool {
	if errChain == nil {
		return false
	}

	hasStacks := false
	isSupersededTrace := false
	throw.Walk(errChain, func(err error, holder throw.StackTraceHolder) bool {
		var localSt throw.StackTrace
		if holder != nil {
			if st, stMode := holder.DeepestStackTrace(); st != nil {
				if !isSupersededTrace {
					hasStacks = true
					localSt = st
				}
				switch {
				case stMode == throw.SupersededTrace:
					isSupersededTrace = true
				case stMode == throw.InheritedTrace:
					localSt = nil
				default:
					isSupersededTrace = false
				}
			}
			//v.appendError(mf, true, holder)
			//v.appendExtraInfo(mf, holder)
			if vv, ok := holder.(throw.PanicHolder); ok {
				v.hasPanic = true
				r := vv.Recovered()
				if _, ok := r.(error); !ok {
					v.appendError(mf, r, nil)
				}
			}
			err = holder.Cause()
		}

		if err == nil {
			return false
		}

		hasReport := v.appendError(mf, err, localSt)
		if !v.appendExtraInfo(mf, err) && !hasReport {
			if s := err.Error(); s != "" {
				if wrapped := errors.Unwrap(err); wrapped != nil {
					s, _ = cutOut(s, wrapped.Error())
				}
				v.reports = append(v.reports, errorReport{nil, s, nil})
			}
		}

		return false
	})

	switch {
	case len(v.reports) > 1:
		for i, e := range v.reports {
			if v.msg == "" && e.strValue != "" {
				v.msg = e.strValue
				v.reports[i].strValue = ""
				if v.stack != nil {
					break
				}
			}
			if v.stack == nil && e.stack != nil && e.stack.IsFullStack() {
				v.stack = e.stack
				v.reports[i].stack = nil
				if v.msg != "" {
					break
				}
			}
		}
	case len(v.reports) == 0:
		// very strange ...
		if ls, ok := errChain.(LogStringer); ok {
			v.msg = ls.LogString()
		} else {
			v.msg = errChain.Error()
		}
		if hasStacks {
			v.msg = throw.TrimStackTrace(v.msg)
		}
	default:
		v.msg = v.reports[0].strValue
		v.stack = v.reports[0].stack
		if v.reports[0].marshaller == nil {
			v.reports = nil
		} else {
			v.reports[0].stack = nil
			v.reports[0].strValue = ""
		}
	}

	return true
}

func cutOut(s, substr string) (string, bool) {
	if substr == "" {
		return s, false
	}
	if i := strings.LastIndex(s, substr); i >= 0 {
		s0 := s[:i]
		s1 := s[i+len(substr):]
		if s0t := strings.TrimSpace(s0); s0t != "" {
			return s0 + "%w" + s1, true
		}
		if s1t := strings.TrimSpace(s1); s1t != "" {
			return "%w" + s1, true
		}
		return "", true
	}
	return s, false
}

func (v errorMarshaller) MarshalLogObject(output LogObjectWriter, collector LogObjectMetricCollector) (string, bool) {
	for i := len(v.reports) - 1; i >= 0; i-- {
		el := v.reports[i]
		if el.marshaller != nil {
			if s, defMessage := el.marshaller.MarshalLogObject(output, collector); !defMessage && s != "" {
				output.AddErrorField(s, el.stack, 0, false)
				el.stack = nil
			}
		}
		if s := el.strValue; s != "" || el.stack != nil {
			output.AddErrorField(s, el.stack, 0, false)
		}
	}
	if v.stack != nil || v.severity != 0 {
		output.AddErrorField("", v.stack, v.severity, v.hasPanic)
	}
	return v.msg, false
}

func (v errorMarshaller) MarshalMutedLogObject(collector LogObjectMetricCollector) {
	if collector == nil {
		return
	}
	for i := len(v.reports) - 1; i >= 0; i-- {
		if mm, ok := v.reports[i].marshaller.(MutedLogObjectMarshaller); ok {
			mm.MarshalMutedLogObject(collector)
		}
	}
}

func (v errorMarshaller) IsEmpty() bool {
	return len(v.reports) == 0
}
