// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

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
	if CmpStateFuncs(expectedStep, actualStep) {
		c.position++
		return nil
	}

	return errors.Errorf("step %d call wrong func (expected %s, got %s)", c.position, getFuncName(expectedStep), getFuncName(actualStep))
}

func CmpStateFuncs(want, got interface{}) bool {
	return reflect.ValueOf(want).Pointer() == reflect.ValueOf(got).Pointer()
}

func CheckWrapper(c *SMStepChecker, t *testing.T) func(smachine.StateFunc) smachine.StateUpdate {
	return func(stateFunc smachine.StateFunc) smachine.StateUpdate {
		require.NoError(t, c.Check(stateFunc))
		return smachine.StateUpdate{}
	}
}
