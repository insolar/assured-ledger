// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

type smTestRestoreStep struct {
	smachine.StateMachineDeclTemplate

	wasContinued bool
}

func (s *smTestRestoreStep) GetInitStateFor(_ smachine.StateMachine) smachine.InitFunc {
	return s.stepInit
}

func (s *smTestRestoreStep) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return s
}

func (s *smTestRestoreStep) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(s.migrate)
	return ctx.Jump(s.stepWaitInfinity)
}

func (s *smTestRestoreStep) stepWaitInfinity(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.wasContinued = true
	return ctx.Sleep().ThenRepeat()
}

func (s *smTestRestoreStep) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	s.wasContinued = false
	return ctx.RestoreStep(ctx.AffectedStep())
}

func TestSlotMachine_RestoreStep(t *testing.T) {
	ctx := instestlogger.TestContext(t)

	helper := newTestsHelper()
	s := smTestRestoreStep{}
	helper.add(ctx, &s)
	require.False(t, s.wasContinued)

	// make 1 iteration
	helper.iter(nil)

	require.True(t, s.wasContinued)

	helper.migrate()
	helper.iter(nil)

	require.False(t, s.wasContinued)
}

