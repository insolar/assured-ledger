// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smsync

import (
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

func NewInfiniteLock(name string) smachine.SyncLink {
	return smachine.NewSyncLink(&infiniteLock{name: name})
}

type infiniteLock struct {
	name  string
	count int32 // atomic
}

func (p *infiniteLock) CheckState() smachine.BoolDecision {
	return false
}

func (p *infiniteLock) UseDependency(dep smachine.SlotDependency, flags smachine.SlotDependencyFlags) smachine.Decision {
	if entry, ok := dep.(*infiniteLockEntry); ok {
		switch {
		case !entry.IsCompatibleWith(flags):
			return smachine.Impossible
		case entry.ctl == p:
			return smachine.NotPassed
		}
	}
	return smachine.Impossible
}

func (p *infiniteLock) ReleaseDependency(dep smachine.SlotDependency) (bool, smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	pd, sl := dep.ReleaseAll()
	return true, nil, pd, sl
}

func (p *infiniteLock) CreateDependency(_ smachine.SlotLink, flags smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
	atomic.AddInt32(&p.count, 1)
	return false, &infiniteLockEntry{p, flags}
}

func (p *infiniteLock) GetLimit() (limit int, isAdjustable bool) {
	return 0, false
}

func (p *infiniteLock) AdjustLimit(int, bool) ([]smachine.StepLink, bool) {
	panic("illegal state")
}

func (p *infiniteLock) GetCounts() (active, inactive int) {
	return 0, int(p.count)
}

func (p *infiniteLock) GetName() string {
	return p.name
}

func (p *infiniteLock) EnumQueues(smachine.EnumQueueFunc) bool {
	return false
}

var _ smachine.SlotDependency = &infiniteLockEntry{}

type infiniteLockEntry struct {
	ctl       *infiniteLock
	slotFlags smachine.SlotDependencyFlags
}

func (v infiniteLockEntry) IsReleaseOnStepping() bool {
	return v.slotFlags&smachine.SyncForOneStep != 0
}

func (infiniteLockEntry) IsReleaseOnWorking() bool {
	return false
}

func (v infiniteLockEntry) ReleaseAll() ([]smachine.PostponedDependency, []smachine.StepLink) {
	atomic.AddInt32(&v.ctl.count, -1)
	return nil, nil
}

func (v infiniteLockEntry) IsCompatibleWith(requiredFlags smachine.SlotDependencyFlags) bool {
	return v.slotFlags.IsCompatibleWith(requiredFlags)
}
