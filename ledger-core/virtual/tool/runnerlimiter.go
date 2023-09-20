package tool

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
)

const RunnerLimiterName = "RunnerLimiter"

func NewRunnerLimiter(semaphoreLimit int) RunnerLimiter {
	return RunnerLimiter{
		semaphore: smsync.NewSemaphoreWithFlags(semaphoreLimit, RunnerLimiterName, smsync.QueueAllowsPriority),
	}
}

type RunnerLimiter struct {
	semaphore smsync.SemaphoreLink
}

func (cs RunnerLimiter) PartialLink() smachine.SyncLink {
	return cs.semaphore.PartialLink()
}

func (cs RunnerLimiter) NewChildSemaphore(childValue int, name string) smsync.SemaChildLink {
	return cs.semaphore.NewChildExt(true, childValue, name, smsync.AllowPartialRelease|smsync.PrioritizePartialAcquire)
}
