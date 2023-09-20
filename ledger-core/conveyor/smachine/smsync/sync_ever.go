package smsync

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewAlwaysOpen(name string) smachine.SyncLink {
	return smachine.NewSyncLink(&openLock{name: name})
}

type openLock struct {
	name  string
}

func (p *openLock) CheckState() smachine.BoolDecision {
	return true
}

func (p *openLock) UseDependency(smachine.SlotDependency, smachine.SlotDependencyFlags) smachine.Decision {
	return smachine.Impossible
}

func (p *openLock) ReleaseDependency(dep smachine.SlotDependency) (bool, smachine.SlotDependency, []smachine.PostponedDependency, []smachine.StepLink) {
	pd, sl := dep.ReleaseAll()
	return true, nil, pd, sl
}

func (p *openLock) CreateDependency(smachine.SlotLink, smachine.SlotDependencyFlags) (smachine.BoolDecision, smachine.SlotDependency) {
	return true, nil
}

func (p *openLock) GetLimit() (limit int, isAdjustable bool) {
	return 1, false
}

func (p *openLock) AdjustLimit(int, bool) ([]smachine.StepLink, bool) {
	panic(throw.IllegalState())
}

func (p *openLock) GetCounts() (active, inactive int) {
	return -1, 0
}

func (p *openLock) GetName() string {
	return p.name
}

func (p *openLock) EnumQueues(smachine.EnumQueueFunc) bool {
	return false
}
