package smachine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

type smReplaceAndMigrate1 struct {
	smachine.StateMachineDeclTemplate

	replaced  bool
	migrated1 bool
	migrated2 bool
	executed  bool
}

func (s *smReplaceAndMigrate1) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *smReplaceAndMigrate1) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smReplaceAndMigrate1) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(func(ctx smachine.MigrationContext) smachine.StateUpdate {
		s.migrated1 = true
		return ctx.Stop()
	})
	return ctx.Jump(s.stepExecute)
}

func (s *smReplaceAndMigrate1) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.replaced = true
	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &smReplaceAndMigrate2{migrated: &s.migrated2, executed: &s.executed}
	})
}

type smReplaceAndMigrate2 struct {
	smachine.StateMachineDeclTemplate

	migrated *bool
	executed *bool
}

func (s *smReplaceAndMigrate2) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.Init
}

func (s *smReplaceAndMigrate2) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smReplaceAndMigrate2) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.JumpExt(smachine.SlotStep{
		Transition: s.stepExecute,
		Migration: func(ctx smachine.MigrationContext) smachine.StateUpdate {
			*s.migrated = true
			return ctx.Stop()
		},
	})
}

func (s *smReplaceAndMigrate2) stepExecute(ctx smachine.ExecutionContext) smachine.StateUpdate {
	*s.executed = true
	return ctx.Stop()
}

func TestSlotMachine_ReplaceAndMigrate(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	helper := newTestsHelper()
	s := smReplaceAndMigrate1{}
	helper.add(ctx, &s)
	helper.iter(func() bool {
		return !s.replaced
	})

	assert.True(t, s.replaced)

	helper.migrate()
	helper.iter(nil)

	assert.False(t, s.migrated1)
	assert.True(t, s.migrated2)
	assert.False(t, s.executed)
}
