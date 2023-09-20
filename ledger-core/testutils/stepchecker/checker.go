package stepchecker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

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
		out.WriteString(convlog.GetStepName(s.fn))
	default:
	}
	out.WriteString("]")
	return out.String()
}

type Checker struct {
	failed   bool
	position int
	steps    []stepDecl
}

func New() *Checker {
	return &Checker{}
}

func (c *Checker) AddStep(step smachine.StateFunc) {
	if step == nil {
		panic("unexpected step: nil")
	}

	c.steps = append(c.steps, stepDecl{
		t:  Jump,
		fn: step,
	})
}

func (c *Checker) AddRepeat() {
	c.steps = append(c.steps, stepDecl{
		t:  Repeat,
		fn: nil,
	})
}

func (c *Checker) CheckJump(actualStep smachine.StateFunc) error {
	if c.failed {
		return throw.IllegalState()
	}
	if c.position > len(c.steps) {
		return fmt.Errorf("unexpected step '%s'", convlog.GetStepName(actualStep))
	}

	expectedStep := c.steps[c.position]
	if expectedStep.t != Jump {
		return fmt.Errorf("unexpected step type 'Jump', got '%s'", expectedStep.t.String())
	}
	if !testutils.CmpStateFuncs(expectedStep.fn, actualStep) {
		return fmt.Errorf("step '%d' call wrong func (expected '%s', got '%s')", c.position, convlog.GetStepName(expectedStep.fn), convlog.GetStepName(actualStep))
	}

	c.position++
	return nil
}

func (c *Checker) CheckJumpW(t *testing.T) func(smachine.StateFunc) smachine.StateUpdate {
	return func(stateFunc smachine.StateFunc) smachine.StateUpdate {
		require.NoError(t, c.CheckJump(stateFunc))
		return smachine.StateUpdate{}
	}
}

func (c *Checker) CheckRepeat() error {
	if c.failed {
		return throw.IllegalState()
	}
	if c.position > len(c.steps) {
		return throw.New("unexpected repeat")
	}

	expectedStep := c.steps[c.position]
	if expectedStep.t != Repeat {
		return fmt.Errorf("unexpected step type 'Repeat', got '%s'", expectedStep.t.String())
	}

	c.position++
	return nil
}

func (c *Checker) CheckRepeatW(t *testing.T) func() smachine.StateUpdate {
	return func() smachine.StateUpdate {
		require.NoError(t, c.CheckRepeat())
		return smachine.StateUpdate{}
	}
}

func (c *Checker) CheckDone() error {
	left := len(c.steps) - c.position
	if left > 0 {
		names := make([]string, left)
		for i := c.position; i < len(c.steps); i++ {
			names[i-c.position] = c.steps[i].String()
		}
		return fmt.Errorf("not all steps are done (%s)", strings.Join(names, ", "))
	}
	return nil
}
