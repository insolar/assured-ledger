// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
)

const RunnerParallelLimiter = "RunnerParallelProcessingLimiter"

func NewParallelProcessingLimiter(semaphoreLimit int) ParallelProcessingLimiter {
	return ParallelProcessingLimiter{
		semaphore: smsync.NewSemaphoreWithFlags(semaphoreLimit, RunnerParallelLimiter, smsync.QueueAllowsPriority),
	}
}

type ParallelProcessingLimiter struct {
	semaphore smsync.SemaphoreLink
}

func (cs ParallelProcessingLimiter) PartialLink() smachine.SyncLink {
	return cs.semaphore.PartialLink()
}

func (cs ParallelProcessingLimiter) NewChildSemaphore(childValue int, name string) smsync.SemaChildLink {
	return cs.semaphore.NewChildExt(true, childValue, name, smsync.AllowPartialRelease|smsync.PrioritizePartialAcquire)
}
