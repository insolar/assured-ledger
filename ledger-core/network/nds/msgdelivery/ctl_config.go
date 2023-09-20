package msgdelivery

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/msgdelivery/retries"
)

type Config struct {
	TimeCycle        time.Duration
	MessageBatchSize uint
	MessageSender    SenderConfig
	StateSender      SenderConfig
}

type SenderConfig struct {
	RetryIntervals [retries.RetryStages]time.Duration

	FastQueue  int
	RetryQueue int

	SenderWorkerConfig
}

type SenderWorkerConfig struct {
	ParallelWorkers        int
	ParallelPeersPerWorker int
	MaxPostponedPerWorker  int
}
