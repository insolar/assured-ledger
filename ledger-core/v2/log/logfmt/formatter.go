// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package logfmt

import (
	"reflect"
)

type FormatFunc func(...interface{}) string
type FormatfFunc func(string, ...interface{}) string

type MsgFormatConfig struct {
	Sformat  FormatFunc
	Sformatf FormatfFunc
	MFactory MarshallerFactory

	TimeFmt string
}

type FieldReporterFunc func(collector LogObjectMetricCollector, fieldName string, v interface{})

type MarshallerFactory interface {
	CreateLogObjectMarshaller(o reflect.Value) LogObjectMarshaller
	RegisterFieldReporter(fieldType reflect.Type, fn FieldReporterFunc)
}

func (v MsgFormatConfig) fmtLogStruct(a interface{}, optionalStruct bool) (LogObjectMarshaller, *string) {
	if vv, ok := a.(LogObject); ok {
		m := vv.GetLogObjectMarshaller()
		if m != nil {
			return m, nil
		}
		vr := reflect.ValueOf(vv)
		if vr.Kind() == reflect.Ptr {
			vr = vr.Elem()
		}
		m = v.MFactory.CreateLogObjectMarshaller(vr)
		if m != nil {
			return m, nil
		}
	}

	switch vv := a.(type) {
	case LogObjectMarshaller:
		return vv, nil
	case string: // the most obvious case
		return nil, &vv
	case *string: // handled separately to avoid unnecessary reflect
		return nil, vv
	case nil:
		return nil, nil
	default:
		if s, t, isNil := prepareValue(vv); t != reflect.Invalid {
			if isNil {
				return nil, nil
			}
			return nil, &s
		}

		vr := reflect.ValueOf(vv)
		switch vr.Kind() {
		case reflect.Ptr:
			if vr.IsNil() {
				break
			}
			vr = vr.Elem()
			if vr.Kind() != reflect.Struct {
				break
			}
			fallthrough
		case reflect.Struct:
			if !optionalStruct || len(vr.Type().PkgPath()) == 0 {
				return v.MFactory.CreateLogObjectMarshaller(vr), nil
			}
		}
	}
	return nil, nil
}

func (v MsgFormatConfig) FmtLogStruct(a interface{}) (LogObjectMarshaller, string) {
	if m, s := v.fmtLogStruct(a, false); m != nil {
		return m, ""
	} else if s != nil {
		return m, *s
	}
	return nil, v.Sformat(a)
}

func (v MsgFormatConfig) FmtLogStructOrObject(a interface{}) (LogObjectMarshaller, string) {
	if m, s := v.fmtLogStruct(a, true); m != nil {
		return m, ""
	} else if s != nil {
		return nil, *s
	}
	return nil, v.Sformat(a)
}

func (v MsgFormatConfig) FmtLogObject(a ...interface{}) string {
	return v.Sformat(a...)
}

func (v MsgFormatConfig) PrepareMutedLogObject(a ...interface{}) LogObjectMarshaller {
	if len(a) != 1 {
		return nil
	}

	switch vv := a[0].(type) {
	case LogObject:
		m := vv.GetLogObjectMarshaller()
		if m != nil {
			return m
		}
		vr := reflect.ValueOf(vv)
		if vr.Kind() == reflect.Ptr {
			vr = vr.Elem()
		}
		return v.MFactory.CreateLogObjectMarshaller(vr)
	case LogObjectMarshaller:
		return vv
	case string, *string, nil: // the most obvious case(s) - avoid reflect
		return nil
	default:
		vr := reflect.ValueOf(vv)
		switch vr.Kind() {
		case reflect.Ptr:
			if vr.IsNil() {
				return nil
			}
			vr = vr.Elem()
			if vr.Kind() != reflect.Struct {
				return nil
			}
			fallthrough
		case reflect.Struct:
			if len(vr.Type().Name()) == 0 { // only unnamed objects are handled by default
				return v.MFactory.CreateLogObjectMarshaller(vr)
			}
		}
	}
	return nil
}

type MsgTemplate struct{}

func (*MsgTemplate) GetLogObjectMarshaller() LogObjectMarshaller {
	return nil
}
