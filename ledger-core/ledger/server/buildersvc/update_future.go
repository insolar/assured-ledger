// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewFuture(name string) *Future {
	return &Future{
		ready: smsync.NewConditionalBool(false, "drop-future" + name),
	}
}

type Future struct {
	ready     smsync.BoolConditionalLink

	mutex       sync.Mutex
	allocations []ledger.DirectoryIndex
	err 	    error
}

func (p *Future) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

func (p *Future) GetFutureAllocation() (isReady bool, allocations []ledger.DirectoryIndex) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	switch {
	case p.err != nil:
		return true, nil
	case p.allocations == nil:
		return false, nil
	default:
		return true, p.allocations
	}
}

func (p *Future) GetFutureResult() (isReady bool, err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case p.err != nil:
		return true, p.err
	case p.allocations == nil:
		return false, nil
	case len(p.allocations) == 0:
		return true, throw.IllegalState()
	default:
		return true, nil
	}
}

func (p *Future) TrySetFutureResult(allocations []ledger.DirectoryIndex, err error) bool {
	switch {
	case err != nil:
		if len(allocations) != 0 {
			panic(throw.IllegalValue())
		}
	case len(allocations) == 0:
		panic(throw.IllegalValue())
	}

	p.mutex.Lock()
	if p.err != nil || p.allocations != nil {
		p.mutex.Unlock()
		return false
	}
	p.allocations, p.err = allocations, err
	p.mutex.Unlock()

	smachine.ApplyAdjustmentAsync(p.ready.NewValue(true))
	return true
}
