// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"runtime"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
)

const RunnerParallelLimiter = "RunnerParallelProcessingLimiter"

func NewParallelProcessingLimiter(maxTasks int) ParallelProcessingLimiter {
	if maxTasks <= 0 {
		maxTasks = runtime.NumCPU() - 2
	}
	if maxTasks < 1 {
		maxTasks = 1
	}
	return ParallelProcessingLimiter{
		semaphore: smsync.NewSemaphoreWithFlags(maxTasks, RunnerParallelLimiter, smsync.QueueAllowsPriority),
	}
}

type ParallelProcessingLimiter struct {
	semaphore smsync.SemaphoreLink
}

func (cs ParallelProcessingLimiter) PartialLink() smachine.SyncLink {
	return cs.semaphore.PartialLink()
}

func (cs ParallelProcessingLimiter) NewHierarchySemaphore(childValue int, name string) smsync.SemaChildLink {
	return cs.semaphore.NewChildExt(true, childValue, name, smsync.AllowPartialRelease|smsync.PrioritizePartialAcquire)
}
