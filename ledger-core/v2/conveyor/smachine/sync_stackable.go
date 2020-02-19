// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

// methods of this interfaces can be protected by mutex
type dependencyStackController interface {
	ReleaseStacked(releasedBy *dependencyQueueEntry)
}

type dependencyStack struct {
	controller dependencyStackController
}

func (p *dependencyStack) PushStack(entry *dependencyQueueEntry, link StepLink) PostponedDependency {
	return nil
}

func (p *dependencyStack) PopStack(entry *dependencyQueueEntry) {
	if p == nil {
		return
	}
	p.controller.ReleaseStacked(entry)
}
