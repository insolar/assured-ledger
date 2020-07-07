// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewFuture(name string) *Future {
	return &Future{
		ready: smsync.NewConditionalBool(false, "drop-future" + name),
	}
}

type Future struct {
	ready     smsync.BoolConditionalLink
	committed atomickit.Uint32
	err 	  error
}

func (p *Future) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

// GetFutureResult can only be used after SyncLink is triggered
func (p *Future) GetFutureResult() (isReady bool, allocationBase uint32, err error) {
	switch v := p.committed.Load(); {
	case v == 0:
		return false, 0, nil
	case v == math.MaxUint32:
		return true, 0, p.err
	default:
		return true, v, nil
	}
}

func (p *Future) TrySetFutureResult(committed bool, allocatedBase uint32, err error) bool {
	switch {
	case allocatedBase == math.MaxUint32:
		panic(throw.IllegalValue())
	case !committed:
		allocatedBase = math.MaxUint32
	case allocatedBase == 0:
		panic(throw.IllegalValue())
	case err != nil:
		panic(throw.IllegalValue())
	}
	if !p.committed.CompareAndSwap(0, allocatedBase) {
		return false
	}
	p.err = err

	smachine.ApplyAdjustmentAsync(p.ready.NewValue(true))
	return true
}
