// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

// methods of this interfaces can be protected by mutex
type dependencyStackController interface {
	ReleaseStacked(releasedBy *dependencyQueueEntry, flags SlotDependencyFlags)
}

type dependencyStackEntry struct {
	controller dependencyStackController
}

func (p *dependencyStackEntry) ActivateStack(activateBy *dependencyQueueEntry, link StepLink) PostponedDependency {
	return nil
}

func (p *dependencyStackEntry) ReleasedBy(entry *dependencyQueueEntry, flags SlotDependencyFlags) {
	if p == nil {
		return
	}
	p.controller.ReleaseStacked(entry, flags)
}
