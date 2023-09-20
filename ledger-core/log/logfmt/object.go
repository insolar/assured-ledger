package logfmt

import (
	"reflect"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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
