// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"sync/atomic"
)

func NewInfiniteLock(name string) SyncLink {
	return NewSyncLink(&infiniteLock{name: name})
}

type infiniteLock struct {
	name  string
	count int32 //atomic
}

func (p *infiniteLock) CheckState() BoolDecision {
	return false
}

func (p *infiniteLock) UseDependency(dep SlotDependency, flags SlotDependencyFlags) Decision {
	if entry, ok := dep.(*infiniteLockEntry); ok {
		switch {
		case !entry.IsCompatibleWith(flags):
			return Impossible
		case entry.ctl == p:
			return NotPassed
		}
	}
	return Impossible
}

func (p *infiniteLock) CreateDependency(_ SlotLink, flags SlotDependencyFlags) (BoolDecision, SlotDependency) {
	atomic.AddInt32(&p.count, 1)
	return false, &infiniteLockEntry{p, flags}
}

func (p *infiniteLock) GetLimit() (limit int, isAdjustable bool) {
	return 0, false
}

func (p *infiniteLock) AdjustLimit(int, bool) ([]StepLink, bool) {
	panic("illegal state")
}

func (p *infiniteLock) GetCounts() (active, inactive int) {
	return 0, int(p.count)
}

func (p *infiniteLock) GetName() string {
	return p.name
}

func (p *infiniteLock) EnumQueues(EnumQueueFunc) bool {
	return false
}

var _ SlotDependency = &infiniteLockEntry{}

type infiniteLockEntry struct {
	ctl       *infiniteLock
	slotFlags SlotDependencyFlags
}

func (v infiniteLockEntry) IsReleaseOnStepping() bool {
	return v.slotFlags&syncForOneStep != 0
}

func (infiniteLockEntry) IsReleaseOnWorking() bool {
	return false
}

func (v infiniteLockEntry) Release() (SlotDependency, []PostponedDependency, []StepLink) {
	v.ReleaseAll()
	return nil, nil, nil
}

func (v infiniteLockEntry) ReleaseAll() ([]PostponedDependency, []StepLink) {
	atomic.AddInt32(&v.ctl.count, -1)
	return nil, nil
}

func (v infiniteLockEntry) IsCompatibleWith(requiredFlags SlotDependencyFlags) bool {
	return v.slotFlags.isCompatibleWith(requiredFlags)
}
