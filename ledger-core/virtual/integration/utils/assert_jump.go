// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/journal"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
)

func AssertNotJumpToStep(t *testing.T, j *journal.Journal, stepName string) {
	j.Subscribe(func(event debuglogger.UpdateEvent) predicate.SubscriberState {
		if event.Update.UpdateType == "jump" {
			convlog.PrepareStepName(&event.Update.NextStep)
			name := event.Update.NextStep.GetStepName()
			assert.NotContains(t, name, stepName, "SM should not jump to step: "+stepName)
		}

		return predicate.RetainSubscriber
	})
}
