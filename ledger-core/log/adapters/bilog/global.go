package bilog

import (
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/log"
)

var gLevel uint32

func getGlobalFilter() log.Level {
	return log.Level(atomic.LoadUint32(&gLevel))
}

func setGlobalFilter(level log.Level) {
	atomic.StoreUint32(&gLevel, uint32(level))
}
