// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"errors"
	"reflect"
)

func asDetail(value interface{}, target interface{}) bool {
	targetType, val := checkAsTarget(target)
	if value == nil {
		return false
	}
	if reflect.TypeOf(value).AssignableTo(targetType) {
		val.Elem().Set(reflect.ValueOf(value))
		return true
	}
	if x, ok := value.(interface{ AsDetail(interface{}) bool }); ok && x.AsDetail(target) {
		return true
	}
	return false
}

func FindDetail(value error, target interface{}) bool {
	targetType, val := checkAsTarget(target)
	for value != nil {
		switch valueType := reflect.TypeOf(value); {
		case valueType.Kind() == reflect.Ptr:
			if valueType.Elem().AssignableTo(targetType) {
				val.Elem().Set(reflect.ValueOf(value).Elem())
				return true
			}
		case valueType.AssignableTo(targetType):
			val.Elem().Set(reflect.ValueOf(value))
			return true
		}
		if x, ok := value.(interface{ As(interface{}) bool }); ok && x.As(target) {
			return true
		}
		if x, ok := value.(interface{ AsDetail(interface{}) bool }); ok && x.AsDetail(target) {
			return true
		}
		value = errors.Unwrap(value)
	}
	return false
}

func checkAsTarget(target interface{}) (reflect.Type, reflect.Value) {
	if target == nil {
		panic("errors: target cannot be nil")
	}
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("errors: target must be a non-nil pointer")
	}
	targetType := typ.Elem()
	switch targetType.Kind() {
	case reflect.Interface, reflect.Struct:
		return targetType, val
	default:
		panic("errors: *target must be interface or struct")
	}
}
