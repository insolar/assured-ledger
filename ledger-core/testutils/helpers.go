// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/reflectkit"
)

func AssertConditionalBuilderJumpStep(t *testing.T, f smachine.StateFunc) func() smachine.ConditionalBuilder {
	return func() (c1 smachine.ConditionalBuilder) {
		t.Helper()

		return smachine.NewStateConditionalBuilderMock(t).ThenJumpMock.Set(AssertJumpStep(t, f))
	}
}

func AssertJumpStep(t *testing.T, expected smachine.StateFunc) func(smachine.StateFunc) smachine.StateUpdate {
	codeExpected := reflectkit.CodeOf(expected)

	return func(got smachine.StateFunc) smachine.StateUpdate {
		t.Helper()

		codeGot := reflectkit.CodeOf(got)
		if codeExpected != codeGot {
			assert.Failf(t, "Failed to compare step Jumps", "expected '%s', got '%s'",
				convlog.GetStepName(expected),
				convlog.GetStepName(got))
		}
		return smachine.StateUpdate{}
	}
}

func AssertMigration(t *testing.T, expected smachine.MigrateFunc) func(smachine.MigrateFunc) {
	codeExpected := reflectkit.CodeOf(expected)

	return func(got smachine.MigrateFunc) {
		t.Helper()

		codeGot := reflectkit.CodeOf(got)
		if codeExpected != codeGot {
			assert.Failf(t, "Failed to compare migrate Jumps", "expected '%s', got '%s'",
				convlog.GetStepName(expected),
				convlog.GetStepName(got))
		}
	}
}
