// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"reflect"
)

type errorMarshaller struct {
	errors   []errorLevel
	stack    throw.StackTrace
	msg      string
	hasPanic bool
}

type errorLevel struct {
	marshaller LogObjectMarshaller
	//stack throw.StackTrace
}

func newErrorMarshaller(errChain error, mf MarshallerFactory) (errorMarshaller, bool) {
	if errChain == nil {
		return errorMarshaller{}, false
	}

	var errs []errorLevel
	var fullStack, anyStack throw.StackTrace
	hasPanic := false

	throw.Walk(errChain, func(err error, holder throw.StackTraceHolder) bool {
		el := errorLevel{}
		cause := err
		var extra interface{}

		switch {
		case err == nil:
			return false
		case holder == nil:
			extra = err
		default:
			if st := holder.ShallowStackTrace(); st != nil {
				anyStack = st
				if fullStack == nil || st.IsFullStack() {
					fullStack = st
				}
			}
			//el.stack = st
			cause = holder.Cause()
			extra = holder
		}

		if cause != nil {
			el.marshaller = mf.CreateLogObjectMarshaller(reflect.ValueOf(cause))
		}

		if vv, ok := extra.(throw.PanicHolder); ok {
			hasPanic = true
			r := vv.Recovered()
			if _, ok := r.(error); ok {
				extra = nil
			} else {
				extra = r
			}
		} else if el.marshaller == nil {
			extra, _ = throw.UnwrapExtraInfo(extra)
		}

		if el.marshaller == nil {
			el.marshaller = mf.CreateLogObjectMarshaller(reflect.ValueOf(extra))
		}

		if el.marshaller != nil {
			errs = append(errs, el)
		}

		return false
	})

	msg := ""
	if ls, ok := errChain.(LogStringer); ok {
		msg = ls.LogString()
	} else {
		msg = errChain.Error()
		if anyStack != nil {
			msg = throw.TrimStackTrace(msg)
		}
	}
	if fullStack == nil {
		fullStack = anyStack
	}

	return errorMarshaller{errs, fullStack, msg, hasPanic}, true
}

func (v errorMarshaller) MarshalLogObject(output LogObjectWriter, collector LogObjectMetricCollector) string {
	for _, el := range v.errors {
		msg := el.marshaller.MarshalLogObject(output, collector)
		if msg != "" {
			output.AddErrorField(msg, nil, false)
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
