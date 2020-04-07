// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"reflect"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// Presence of this interface indicates that this object can be used as a log event
type LogObject interface {
	// should return nil to use default (external) marshaller
	GetLogObjectMarshaller() LogObjectMarshaller
}

type LogObjectMarshaller interface {
	MarshalLogObject(LogObjectWriter, LogObjectMetricCollector) (msg string, defMsg bool)
}

type LogFieldMarshaller interface {
	MarshalLogFields(LogObjectWriter)
}

type MutedLogObjectMarshaller interface {
	MarshalMutedLogObject(LogObjectMetricCollector)
}

type LogObjectMetricCollector interface {
	LogObjectMetricCollector()
	//ReportMetricSample(metricType uint32, reporterFieldName string, value interface{})
}

type LogFieldFormat struct {
	Fmt    string
	Kind   reflect.Kind
	HasFmt bool
}

func (f LogFieldFormat) IsInt() bool {
	return f.Kind >= reflect.Int && f.Kind <= reflect.Int64
}

func (f LogFieldFormat) IsUint() bool {
	return f.Kind >= reflect.Uint && f.Kind <= reflect.Uintptr
}

type LogObjectWriter interface {
	AddIntField(key string, v int64, fmt LogFieldFormat)
	AddUintField(key string, v uint64, fmt LogFieldFormat)
	AddBoolField(key string, v bool, fmt LogFieldFormat)
	AddFloatField(key string, v float64, fmt LogFieldFormat)
	AddComplexField(key string, v complex128, fmt LogFieldFormat)
	AddStrField(key string, v string, fmt LogFieldFormat)
	AddIntfField(key string, v interface{}, fmt LogFieldFormat)
	AddTimeField(key string, v time.Time, fmt LogFieldFormat)
	AddRawJSONField(key string, v interface{}, fmt LogFieldFormat)
	AddErrorField(msg string, stack throw.StackTrace, severity throw.Severity, hasPanic bool)
}

type LogObjectFields struct {
	Msg    string
	Fields map[string]interface{}
}

func (v LogObjectFields) MarshalLogObject(w LogObjectWriter, _ LogObjectMetricCollector) (string, bool) {
	for k, v := range v.Fields {
		w.AddIntfField(k, v, LogFieldFormat{})
	}
	return v.Msg, false
}

func (v LogObjectFields) MarshalLogFields(w LogObjectWriter) {
	FieldMapMarshaller(v.Fields).MarshalLogFields(w)
}

type FieldMapMarshaller map[string]interface{}

func (v FieldMapMarshaller) MarshalLogFields(w LogObjectWriter) {
	for k, v := range v {
		w.AddIntfField(k, v, LogFieldFormat{})
	}
}
