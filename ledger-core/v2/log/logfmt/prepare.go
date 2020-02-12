//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package logfmt

import (
	"fmt"
	"reflect"
)

type LogStringer interface {
	LogString() string
}

type valuePrepareFn func(interface{}) (string, reflect.Kind, bool)

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()
var logStringerType = reflect.TypeOf((*LogStringer)(nil)).Elem()
var strFuncType = reflect.TypeOf((func() string)(nil))

// WARNING! Sequence of types must match in both findValuePrepareFn() and prepareValue()
// this is a bit slower vs array scan, but may help the compiler with escape analysis
func findPrepareValueFn(t reflect.Type) valuePrepareFn {
	switch {
	case t.AssignableTo(logStringerType):
		return func(value interface{}) (string, reflect.Kind, bool) {
			switch vv := value.(LogStringer); {
			case vv != nil:
				v := vv.LogString()
				return v, reflect.Interface, false
			default:
				return "", reflect.Interface, true
			}
		}
	case t.AssignableTo(strFuncType):
		return func(value interface{}) (string, reflect.Kind, bool) {
			switch vv := value.(func() string); {
			case vv != nil:
				v := vv()
				return v, reflect.Func, false
			default:
				return "", reflect.Func, true
			}
		}
	case t.AssignableTo(errorType):
		return func(value interface{}) (string, reflect.Kind, bool) {
			switch vv := value.(error); {
			case vv != nil:
				v := vv.Error()
				return v, reflect.Interface, false
			default:
				return "", reflect.Interface, true
			}
		}
	case t.AssignableTo(stringerType):
		return func(value interface{}) (string, reflect.Kind, bool) {
			switch vv := value.(fmt.Stringer); {
			case vv != nil:
				v := vv.String()
				return v, reflect.Interface, false
			default:
				return "", reflect.Interface, true
			}
		}
	default:
		return nil
	}
}

// WARNING! Sequence of types must match in both findValuePrepareFn() and prepareValue()
func prepareValue(iv interface{}) (string, reflect.Kind, bool) {
	switch vv := iv.(type) {
	case nil:
		return "", reflect.Interface, true
	case LogStringer:
		return vv.LogString(), reflect.Interface, false
	case func() string:
		if vv == nil {
			return "", reflect.Func, true
		}
		return vv(), reflect.Func, false
	case error:
		return vv.Error(), reflect.Interface, false
	case fmt.Stringer:
		return vv.String(), reflect.Interface, false
	}
	return "", reflect.Invalid, false
}
