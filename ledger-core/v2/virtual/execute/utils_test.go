// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"reflect"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
)

type SMStepChecker struct {
	position int
	steps    []smachine.StateFunc
}

func NewSMStepChecker() *SMStepChecker {
	return &SMStepChecker{}
}

func (c *SMStepChecker) AddStep(step smachine.StateFunc) {
	if step == nil {
		panic("unexpected step: nil")
	}

	c.steps = append(c.steps, step)
}

func getFuncName(step smachine.StateFunc) string {
	if step == nil {
		panic("unexpected step: nil")
	}

	return runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name()
}

func (c *SMStepChecker) Check(actualStep smachine.StateFunc) error {
	if c.position > len(c.steps) {
		return errors.Errorf("unexpected step %s", getFuncName(actualStep))
	}

	expectedStep := c.steps[c.position]
	if reflect.ValueOf(actualStep).Pointer() == reflect.ValueOf(expectedStep).Pointer() {
		c.position++
		return nil
	}

	return errors.Errorf("step %d call wrong func (expected %s, got %s)", c.position, getFuncName(expectedStep), getFuncName(actualStep))
}

type SharedDataAccessorWrapper struct {
	a *smachine.SharedDataAccessor
}

func NewSharedDataAccessorWrapper(a *smachine.SharedDataAccessor) SharedDataAccessorWrapper {
	return SharedDataAccessorWrapper{a: a}
}

func getFieldAsInterface(val reflect.Value, fieldPos int) interface{} {
	field := val.Field(fieldPos)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func (w SharedDataAccessorWrapper) getLink() *smachine.SharedDataLink {
	return getFieldAsInterface(reflect.ValueOf(w.a).Elem(), 0).(*smachine.SharedDataLink)
}

func (w SharedDataAccessorWrapper) getAccessFn() smachine.SharedDataFunc {
	reflectValue := reflect.ValueOf(w.a).Elem()
	return getFieldAsInterface(reflectValue, 1).(smachine.SharedDataFunc)
}

func (w SharedDataAccessorWrapper) getAccessFnData() interface{} {
	reflectValue := reflect.ValueOf(w.getLink()).Elem()
	return getFieldAsInterface(reflectValue, 1)
}

func CallSharedDataAccessor(s1 smachine.SharedDataAccessor) smachine.SharedAccessReport {
	w := NewSharedDataAccessorWrapper(&s1)
	w.getAccessFn()(w.getAccessFnData())

	return smachine.SharedSlotLocalAvailable
}
