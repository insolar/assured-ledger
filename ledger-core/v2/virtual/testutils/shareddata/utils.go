// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package shareddata

import (
	"reflect"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
)

type AccessorWrapper struct {
	a *smachine.SharedDataAccessor
}

func NewSharedDataAccessorWrapper(a *smachine.SharedDataAccessor) AccessorWrapper {
	return AccessorWrapper{a: a}
}

func getFieldAsInterface(val reflect.Value, fieldPos int) interface{} {
	field := val.Field(fieldPos)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func (w AccessorWrapper) getLink() *smachine.SharedDataLink {
	return getFieldAsInterface(reflect.ValueOf(w.a).Elem(), 0).(*smachine.SharedDataLink)
}

func (w AccessorWrapper) getAccessFn() smachine.SharedDataFunc {
	reflectValue := reflect.ValueOf(w.a).Elem()
	return getFieldAsInterface(reflectValue, 1).(smachine.SharedDataFunc)
}

func (w AccessorWrapper) getAccessFnData() interface{} {
	reflectValue := reflect.ValueOf(w.getLink()).Elem()
	return getFieldAsInterface(reflectValue, 1)
}

func CallSharedDataAccessor(s1 smachine.SharedDataAccessor) smachine.SharedAccessReport {
	w := NewSharedDataAccessorWrapper(&s1)
	w.getAccessFn()(w.getAccessFnData())

	return smachine.SharedSlotLocalAvailable
}
