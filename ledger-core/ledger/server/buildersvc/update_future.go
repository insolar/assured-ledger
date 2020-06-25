// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
<<<<<<< HEAD
<<<<<<< HEAD
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
}

func (p *Future) GetReadySync() smachine.SyncLink {
	return p.ready.SyncLink()
}

func (p *Future) GetUpdateStatus() (isReady bool, allocationBase uint32) {
	switch v := p.committed.Load(); {
	case v == 0:
		return false, 0
	case v == math.MaxUint32:
		return true, 0
	default:
		return true, v
	}
}

func (p *Future) GetCommitStatus() (isReady, isCommitted bool) {
	switch v := p.committed.Load(); {
	case v == 0:
		return false, false
	case v == math.MaxUint32:
		return true, false
	default:
		return true, true
	}
}

func (p *Future) SetCommitted(committed bool, allocatedBase uint32) {
	if !p.TrySetCommitted(committed, allocatedBase) {
		panic(throw.IllegalState())
	}
}

func (p *Future) TrySetCommitted(committed bool, allocatedBase uint32) bool {
	switch {
	case allocatedBase == math.MaxUint32:
		panic(throw.IllegalValue())
	case allocatedBase == 0:
		panic(throw.IllegalValue())
	case !committed:
		allocatedBase = math.MaxUint32
	}
	if !p.committed.CompareAndSwap(0, allocatedBase) {
		return false
	}

	smachine.ApplyAdjustmentAsync(p.ready.NewValue(true))
	return true
=======
=======
>>>>>>> SMs
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type Future struct {

}

func (p *Future) GetReadySync() smachine.SyncLink {

}

func (p *Future) IsCommitted() bool {

<<<<<<< HEAD
>>>>>>> SMs
=======
>>>>>>> SMs
}
