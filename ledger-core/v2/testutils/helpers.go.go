// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/reflectkit"
)

func AssertJumpStep(t *testing.T, f smachine.StateFunc) func(smachine.StateFunc) smachine.StateUpdate {
	return func(s1 smachine.StateFunc) smachine.StateUpdate {
		assert.Equal(t, reflectkit.CodeOf(f), reflectkit.CodeOf(s1))
		return smachine.StateUpdate{}
	}
}

func AssertMigration(t *testing.T, f smachine.MigrateFunc) func(smachine.MigrateFunc) {
	return func(s1 smachine.MigrateFunc) {
		assert.Equal(t, reflectkit.CodeOf(f), reflectkit.CodeOf(s1))
	}
}
