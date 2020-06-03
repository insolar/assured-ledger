// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sworker

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

//var _ smachine.AttachableSlotWorker = &AttachableWorker{}

type AttachableWorker struct {
	// signalSource *synckit.VersionedSignal
}

func (p *AttachableWorker) AttachTo(_ *smachine.SlotMachine, loopLimit uint32, fn smachine.AttachedFunc) (wasDetached bool) {
	//w := &SlotWorker{parent: p, outerSignal: p.signalSource.Mark(), loopLimit: loopLimit}
	//fn(w)
	return false
}

//var _ smachine.FixedSlotWorker = &SlotWorker{}

// nolint:unused,structcheck
type SlotWorker struct {
	// parent      *AttachableWorker
	outerSignal *synckit.SignalVersion
	loopLimit   uint32
}

func (p *SlotWorker) HasSignal() bool {
	return p.outerSignal != nil && p.outerSignal.HasSignal()
}

func (*SlotWorker) IsDetached() bool {
	return false
}

func (p *SlotWorker) GetSignalMark() *synckit.SignalVersion {
	return p.outerSignal
}

func (p *SlotWorker) OuterCall(*smachine.SlotMachine, smachine.NonDetachableFunc) (wasExecuted bool) {
	return false
}

func (p *SlotWorker) DetachableCall(fn smachine.DetachableFunc) (wasDetached bool) {
	//fn(&DetachableSimpleSlotWorker{p})
	return false
}
