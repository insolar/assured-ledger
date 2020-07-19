// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smsync

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type semaPartial struct {
	*semaphoreSync
}

func (v semaPartial) CreateDependency(smachine.SlotLink, smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
	panic(throw.FailHere("Partial semaphore can't be acquired directly"))
}

func (v semaPartial) AdjustLimit(limit int, absolute bool) ([]smachine.StepLink, bool) {
	panic(throw.FailHere("Partial semaphore can't be adjusted"))
}

func (v semaPartial) GetName() string {
	return "partial-" + v.semaphoreSync.GetName()
}
