// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reflectkit

import (
	"reflect"
	"unsafe"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit.TypedReceiver -o ./mocks -s _mock.go -g
type TypedReceiver interface {
	ReceiveBool(reflect.Kind, bool)
	ReceiveInt(reflect.Kind, int64)
	ReceiveUint(reflect.Kind, uint64)
	ReceiveFloat(reflect.Kind, float64)
	ReceiveComplex(reflect.Kind, complex128)
	ReceiveString(reflect.Kind, string)
	ReceiveZero(reflect.Kind)
	ReceiveNil(reflect.Kind)
	ReceiveIface(reflect.Kind, interface{})
	ReceiveElse(t reflect.Kind, v interface{}, isZero bool)
}

type FieldGetterFunc func(reflect.Value) reflect.Value
type ValueToReceiverFunc func(value reflect.Value, out TypedReceiver)
type IfaceToReceiverFunc func(value interface{}, k reflect.Kind, out TypedReceiver)
type IfaceToReceiverFactoryFunc func(t reflect.Type, checkZero bool) IfaceToReceiverFunc
type ValueToReceiverFactoryFunc func(unexported bool, t reflect.Type, checkZero bool) (bool, ValueToReceiverFunc)

func ValueToReceiverFactory(k reflect.Kind, custom IfaceToReceiverFactoryFunc) ValueToReceiverFactoryFunc {
	switch k {
	// ======== Simple values, are safe to read from unexported fields ============
	case reflect.Bool:
		return func(_ bool, _ reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			return false, func(value reflect.Value, outFn TypedReceiver) {
				v := value.Bool()
				if !checkZero || v {
					outFn.ReceiveBool(value.Kind(), v)
				} else {
					outFn.ReceiveZero(value.Kind())
				}
			}
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(_ bool, _ reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			return false, func(value reflect.Value, outFn TypedReceiver) {
				v := value.Int()
				if !checkZero || v != 0 {
					outFn.ReceiveInt(value.Kind(), v)
				} else {
					outFn.ReceiveZero(value.Kind())
				}
			}
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(_ bool, _ reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			return false, func(value reflect.Value, outFn TypedReceiver) {
				v := value.Uint()
				if !checkZero || v != 0 {
					outFn.ReceiveUint(value.Kind(), v)
				} else {
					outFn.ReceiveZero(value.Kind())
				}
			}
		}

	case reflect.Float32, reflect.Float64:
		return func(_ bool, _ reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			return false, func(value reflect.Value, outFn TypedReceiver) {
				v := value.Float()
				if !checkZero || v != 0 {
					outFn.ReceiveFloat(value.Kind(), v)
				} else {
					outFn.ReceiveZero(value.Kind())
				}
			}
		}

	case reflect.Complex64, reflect.Complex128:
		return func(_ bool, _ reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			return false, func(value reflect.Value, outFn TypedReceiver) {
				v := value.Complex()
				if !checkZero || v != 0 {
					outFn.ReceiveComplex(value.Kind(), v)
				} else {
					outFn.ReceiveZero(value.Kind())
				}
			}
		}

	case reflect.String:
		return func(_ bool, _ reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			return false, func(value reflect.Value, outFn TypedReceiver) {
				v := value.String()
				if !checkZero || v != "" {
					outFn.ReceiveString(value.Kind(), v)
				} else {
					outFn.ReceiveZero(value.Kind())
				}
			}
		}

	// ============ Types that require special handling for unexported fields ===========

	// nillable types fully defined at compile time
	case reflect.Func, reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan: //, reflect.UnsafePointer:
		return func(unexported bool, t reflect.Type, checkZero bool) (bool, ValueToReceiverFunc) {
			if custom != nil {
				if fn := custom(t, checkZero); fn != nil {
					return unexported, func(value reflect.Value, outFn TypedReceiver) {
						if value.IsNil() {
							outFn.ReceiveNil(k)
						} else {
							fn(value.Interface(), value.Kind(), outFn)
						}
					}
				}
			}
			return unexported, func(value reflect.Value, outFn TypedReceiver) {
				if value.IsNil() { // avoids interface nil
					outFn.ReceiveNil(k)
				} else {
					outFn.ReceiveElse(value.Kind(), value.Interface(), false)
				}
			}
		}

	// nillable types fully undefined at compile time
	case reflect.Interface:
		return func(unexported bool, t reflect.Type, checkZero bool) (b bool, getterFunc ValueToReceiverFunc) {
			if custom != nil {
				if fn := custom(t, checkZero); fn != nil {
					return unexported, func(value reflect.Value, outFn TypedReceiver) {
						v := value.Interface()
						fn(v, value.Elem().Kind(), outFn)
					}
				}
			}

			return unexported, func(value reflect.Value, outFn TypedReceiver) {
				if value.IsNil() {
					outFn.ReceiveNil(value.Kind())
				} else {
					v := value.Interface()
					outFn.ReceiveElse(value.Kind(), v, false)
				}
			}
		}

	// non-nillable
	case reflect.Struct, reflect.Array:
		return func(unexported bool, t reflect.Type, checkZero bool) (b bool, getterFunc ValueToReceiverFunc) {
			valueFn := custom(t, checkZero)
			if checkZero {
				zeroValue := reflect.Zero(t).Interface()
				return unexported, func(value reflect.Value, outFn TypedReceiver) {
					if v := value.Interface(); valueFn == nil {
						outFn.ReceiveElse(value.Kind(), v, v == zeroValue)
					} else {
						valueFn(v, value.Kind(), outFn)
					}
				}
			}
			return unexported, func(value reflect.Value, outFn TypedReceiver) {
				if v := value.Interface(); valueFn == nil {
					outFn.ReceiveElse(value.Kind(), v, false)
				} else {
					valueFn(v, value.Kind(), outFn)
				}
			}
		}

	// ============ Excluded ===================
	// reflect.UnsafePointer
	default:
		return nil
	}
}

func MakeAddressable(value reflect.Value) reflect.Value {
	if value.CanAddr() {
		return value
	}
	valueCopy := reflect.New(value.Type()).Elem()
	valueCopy.Set(value)
	return valueCopy
}

func FieldValueGetter(index int, fd reflect.StructField, useAddr bool, baseOffset uintptr) FieldGetterFunc {
	if !useAddr {
		return func(value reflect.Value) reflect.Value {
			return value.Field(index)
		}
	}

	fieldOffset := fd.Offset + baseOffset
	fieldType := fd.Type

	return func(value reflect.Value) reflect.Value {
		return offsetFieldGetter(value, fieldOffset, fieldType)
	}
}

func offsetFieldGetter(v reflect.Value, fieldOffset uintptr, fieldType reflect.Type) reflect.Value {
	// base has to be here as a separate variable because of new checkptr check in go 1.14
	// By default it is enabled when -race flag is present.
	// It checks the following: If the result of pointer arithmetic points into
	// a Go heap object, one of the unsafe.Pointer-typed operands must point
	// into the same object.
	// See go release notes for details: https://golang.org/doc/go1.14#compiler
	base := unsafe.Pointer(v.UnsafeAddr())
	return reflect.NewAt(fieldType, unsafe.Pointer(uintptr(base)+fieldOffset)).Elem()
}
