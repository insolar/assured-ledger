package smachine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

type testSM struct {
	smachine.StateMachineDeclTemplate

	ok    bool
	notOk bool
}

func (s *testSM) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *testSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *testSM) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.JumpExt(smachine.SlotStep{
		Transition: func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			s.ok = true
			return ctx.Stop()
		},
		Migration: func(ctx smachine.MigrationContext) smachine.StateUpdate {
			s.notOk = true
			return ctx.Stop()
		},
	})
}

func TestSlotMachine_AddSMAndMigrate(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	helper := newTestsHelper()
	s := testSM{}
	helper.add(ctx, &s)
	helper.migrate()
	helper.iter(nil)

	assert.True(t, s.ok)
	assert.False(t, s.notOk)
}
