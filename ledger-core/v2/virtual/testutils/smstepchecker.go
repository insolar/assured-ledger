// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func CmpStateFuncs(want, got interface{}) bool {
	return reflect.ValueOf(want).Pointer() == reflect.ValueOf(got).Pointer()
}

//go:generate stringer -type=StepDeclType
type StepDeclType int

const (
	Undefined StepDeclType = iota
	Jump
	Repeat
)

type stepDecl struct {
	t  StepDeclType
	fn smachine.StateFunc
}

func (s stepDecl) String() string {
	out := strings.Builder{}
	out.WriteByte('[')
	out.WriteString(s.t.String())
	switch s.t {
	case Jump:
		out.WriteString(", ")
		out.WriteString(getFuncName(s.fn))
	default:
	}
	out.WriteString("]")
	return out.String()
}

type SMStepChecker struct {
	failed   bool
	position int
	steps    []stepDecl
}

func NewSMStepChecker() *SMStepChecker {
	return &SMStepChecker{}
}

func (c *SMStepChecker) AddStep(step smachine.StateFunc) {
	if step == nil {
		panic("unexpected step: nil")
	}

	c.steps = append(c.steps, stepDecl{
		t:  Jump,
		fn: step,
	})
}

func (c *SMStepChecker) AddRepeat() {
	c.steps = append(c.steps, stepDecl{
		t:  Repeat,
		fn: nil,
	})
}

func getFuncName(step smachine.StateFunc) string {
	if step == nil {
		panic("unexpected step: nil")
	}

	fullName := runtime.FuncForPC(reflect.ValueOf(step).Pointer()).Name()
	if lastIndex := strings.LastIndex(fullName, "."); lastIndex >= 0 {
		fullName = fullName[lastIndex+1:]
	}
	if lastIndex := strings.LastIndex(fullName, "-"); lastIndex >= 0 {
		fullName = fullName[:lastIndex]
	}

	return fullName
}

func (c *SMStepChecker) CheckJump(actualStep smachine.StateFunc) error {
	if c.failed {
		return throw.IllegalState()
	}
	if c.position > len(c.steps) {
		return errors.Errorf("unexpected step '%s'", getFuncName(actualStep))
	}

	expectedStep := c.steps[c.position]
	if expectedStep.t != Jump {
		return errors.Errorf("unexpected step type 'Jump', got '%s'", expectedStep.t.String())
	}
	if !CmpStateFuncs(expectedStep.fn, actualStep) {
		return errors.Errorf("step '%d' call wrong func (expected '%s', got '%s')", c.position, getFuncName(expectedStep.fn), getFuncName(actualStep))
	}

	c.position++
	return nil
}

func (c *SMStepChecker) CheckJumpW(t *testing.T) func(smachine.StateFunc) smachine.StateUpdate {
	return func(stateFunc smachine.StateFunc) smachine.StateUpdate {
		require.NoError(t, c.CheckJump(stateFunc))
		return smachine.StateUpdate{}
	}
}

func (c *SMStepChecker) CheckRepeat() error {
	if c.failed {
		return throw.IllegalState()
	}
	if c.position > len(c.steps) {
		return errors.Errorf("unexpected repeat")
	}

	expectedStep := c.steps[c.position]
	if expectedStep.t != Repeat {
		return errors.Errorf("unexpected step type 'Repeat', got '%s'", expectedStep.t.String())
	}

	c.position++
	return nil
}

func (c *SMStepChecker) CheckRepeatW(t *testing.T) func() smachine.StateUpdate {
	return func() smachine.StateUpdate {
		require.NoError(t, c.CheckRepeat())
		return smachine.StateUpdate{}
	}
}

func (c *SMStepChecker) CheckDone() error {
	left := len(c.steps) - c.position
	if left > 0 {
		names := make([]string, left)
		for i := c.position; i < len(c.steps); i++ {
			names[i-c.position] = c.steps[i].String()
		}
		return errors.Errorf("not all steps are done (%s)", strings.Join(names, ", "))
	}
	return nil
}
