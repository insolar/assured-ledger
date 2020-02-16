// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"errors"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"strings"
)

type errorMarshaller struct {
	errors   []errorLevel
	stack    throw.StackTrace
	msg      string
	hasPanic bool
}

type errorLevel struct {
	marshaller LogObjectMarshaller
	strValue   string
	pureError  bool
}

func (v *errorMarshaller) appendErrorLevel(m LogObjectMarshaller, s string, ps *string) bool {
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
		v.errors = append(v.errors, errorLevel{m, s, false})
		return true
	}
	return false
}

func (v *errorMarshaller) appendError(mf MarshallerFactory, isHolder bool, x interface{}) bool {
	m, ps := fmtLogStruct(x, mf, true, true)
	if isHolder {
		ps = nil
	}
	return v.appendErrorLevel(m, "", ps)
}

func (v *errorMarshaller) appendExtraInfo(mf MarshallerFactory, x interface{}) bool {
	if extras, extra, ok := throw.UnwrapExtraInfo(x); ok {
		m, ps := fmtLogStruct(extra, mf, true, true)
		if v.appendErrorLevel(m, extras, ps) {
			return true
		}
		if err, ok := extra.(error); ok {
			return v.appendErrorLevel(nil, err.Error(), nil)
		}
	}
	return false
}

func (v *errorMarshaller) fillLevels(errChain error, mf MarshallerFactory) bool {
	if errChain == nil {
		return false
	}

	var (
		anyStack throw.StackTrace
	)

	throw.Walk(errChain, func(err error, holder throw.StackTraceHolder) bool {
		if holder != nil {
			if st := holder.ShallowStackTrace(); st != nil {
				anyStack = st
				if st.IsFullStack() {
					v.stack = st
				}
			}

			v.appendError(mf, true, holder)
			v.appendExtraInfo(mf, holder)

			if vv, ok := holder.(throw.PanicHolder); ok {
				v.hasPanic = true
				r := vv.Recovered()
				if _, ok := r.(error); !ok {
					v.appendError(mf, false, r)
				}
			}

			err = holder.Cause()
		}

		if err == nil {
			return false
		}

		hasLevel := v.appendError(mf, false, err)
		if !v.appendExtraInfo(mf, err) && !hasLevel {
			if s := err.Error(); s != "" {
				if wrapped := errors.Unwrap(err); wrapped != nil {
					s, _ = cutOut(s, wrapped.Error())
				}
				v.errors = append(v.errors, errorLevel{nil, s, true})
			}
		}

		return false
	})

	if v.stack == nil {
		v.stack = anyStack
	}

	switch {
	case len(v.errors) > 1:
		v.cleanupMessages()
	case len(v.errors) == 0:
		// very strange ...
		if ls, ok := errChain.(LogStringer); ok {
			v.msg = ls.LogString()
		} else {
			v.msg = errChain.Error()
		}
	default:
		v.msg = v.errors[0].strValue
		if v.errors[0].marshaller == nil {
			v.errors = nil
		} else {
			v.errors[0].strValue = ""
		}
	}

	if v.stack != nil {
		v.msg = throw.TrimStackTrace(v.msg)
	}

	return true
}

func cutOut(s, substr string) (string, bool) {
	if substr == "" {
		return s, false
	}
	if i := strings.Index(s, substr); i >= 0 {
		s0 := strings.TrimSpace(s[:i])
		switch s1 := strings.TrimSpace(s[i+len(substr):]); {
		case s1 == "":
			//
		case s0 == "":
			s0 = s1
		default:
			s0 += " " + s1
		}
		return s0, true
	}
	return s, false
}

func (v *errorMarshaller) cleanupMessages() {
	lastMsgIdx := len(v.errors) - 1
	lastMsg := v.errors[lastMsgIdx].strValue
	//if v.stack != nil {
	//	v.errors[lastMsgIdx].strValue = throw.TrimStackTrace(lastMsg)
	//}

	for i := lastMsgIdx - 1; i >= 0; i-- {
		thisMsg := v.errors[i].strValue
		switch {
		case thisMsg == "":
			continue
		case !v.errors[i].pureError:
			if s, ok := cutOut(thisMsg, lastMsg); ok {
				if v.stack != nil {
					s = throw.TrimStackTrace(s)
				}
				v.errors[i].strValue = s
				if s != "" {
					lastMsgIdx = i
				}
				lastMsg = thisMsg
				continue
			} else if v.stack != nil {
				v.errors[i].strValue = throw.TrimStackTrace(thisMsg)
			}
		}
		lastMsgIdx = i
		lastMsg = thisMsg
	}
	v.msg = v.errors[lastMsgIdx].strValue
	v.errors[lastMsgIdx].strValue = ""
}

func (v errorMarshaller) MarshalLogObject(output LogObjectWriter, collector LogObjectMetricCollector) string {
	for _, el := range v.errors {
		if el.marshaller != nil {
			if s := el.marshaller.MarshalLogObject(output, collector); s != "" {
				output.AddErrorField(s, nil, false)
			}
		}
		if s := el.strValue; s != "" {
			output.AddErrorField(s, nil, false)
		}
	}
	if v.stack != nil {
		output.AddErrorField("", v.stack, v.hasPanic)
	}
	return v.msg
}

func (v errorMarshaller) MarshalMutedLogObject(collector LogObjectMetricCollector) {
	if collector == nil {
		return
	}
	for _, el := range v.errors {
		if mm, ok := el.marshaller.(MutedLogObjectMarshaller); ok {
			mm.MarshalMutedLogObject(collector)
		}
	}
}

func (v errorMarshaller) IsEmpty() bool {
	return len(v.errors) == 0
}
