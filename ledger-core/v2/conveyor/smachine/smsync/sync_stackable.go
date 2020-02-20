// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smsync

import "github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"

// methods of this interfaces can be protected by mutex
type dependencyStackController interface {
	containsInStack(q *dependencyQueueHead, entry *dependencyQueueEntry) smachine.Decision
	popStack(q *dependencyQueueHead, entry *dependencyQueueEntry) (smachine.PostponedDependency, *dependencyQueueEntry)
}

type dependencyStack struct {
	controller dependencyStackController
}

func (p *dependencyStack) popStack(q *dependencyQueueHead, entry *dependencyQueueEntry) (smachine.PostponedDependency, *dependencyQueueEntry) {
	if p == nil || p.controller == nil {
		return nil, nil
	}
	return p.controller.popStack(q, entry)
}

func (p *dependencyStack) containsInStack(q *dependencyQueueHead, entry *dependencyQueueEntry) smachine.Decision {
	if p == nil || p.controller == nil {
		return smachine.Impossible
	}
	return p.controller.containsInStack(q, entry)
}
