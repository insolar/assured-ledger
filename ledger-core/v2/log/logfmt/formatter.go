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
	switch vv := a.(type) {
	case LogObject:
		m := vv.GetLogObjectMarshaller()
		if m != nil {
			return m, nil
		}
		vr := reflect.ValueOf(vv)
		if vr.Kind() == reflect.Ptr {
			vr = vr.Elem()
		}
		return v.MFactory.CreateLogObjectMarshaller(vr), nil
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
