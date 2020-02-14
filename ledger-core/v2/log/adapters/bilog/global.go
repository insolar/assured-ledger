// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bilog

import (
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/v2/log"
)

var gLevel uint32

func getGlobalFilter() log.Level {
	return log.Level(atomic.LoadUint32(&gLevel))
}

func setGlobalFilter(level log.Level) {
	atomic.StoreUint32(&gLevel, uint32(level))
}
