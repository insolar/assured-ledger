package smachine

import (
	"context"
)

var _ SlotMachineHolder = slotMachineHolder{}
type slotMachineHolder struct {
	m *SlotMachine
}

func (v slotMachineHolder) GetMachineID() string {
	return v.m.GetMachineID()
}

func (v slotMachineHolder) FindDependency(id string) (interface{}, bool) {
	return v.m.FindDependency(id)
}

func (v slotMachineHolder) PutDependency(id string, vi interface{}) {
	v.m.PutDependency(id, vi)
}

func (v slotMachineHolder) TryPutDependency(id string, vi interface{}) bool {
	return v.m.TryPutDependency(id, vi)
}

func (v slotMachineHolder) AddDependency(vi interface{}) {
	v.m.AddDependency(vi)
}

func (v slotMachineHolder) AddInterfaceDependency(vi interface{}) {
	v.m.AddInterfaceDependency(vi)
}

func (v slotMachineHolder) GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder) {
	return v.m.GetPublishedGlobalAliasAndBargeIn(key)
}

func (v slotMachineHolder) AddNew(ctx context.Context, sm StateMachine, values CreateDefaultValues) (SlotLink, bool) {
	return v.m.AddNew(ctx, sm, values)
}

func (v slotMachineHolder) AddNewByFunc(ctx context.Context, createFunc CreateFunc, values CreateDefaultValues) (SlotLink, bool) {
	return v.m.AddNewByFunc(ctx, createFunc, values)
}

func (v slotMachineHolder) OccupiedSlotCount() int {
	return v.m.OccupiedSlotCount()
}

func (v slotMachineHolder) AllocatedSlotCount() int {
	return v.m.AllocatedSlotCount()
}

func (v slotMachineHolder) ScheduleCall(fn MachineCallFunc, isSignal bool) bool {
	return v.m.ScheduleCall(fn, isSignal)
}

func (v slotMachineHolder) Stop() bool {
	return v.m.Stop()
}

func (v slotMachineHolder) GetStoppingSignal() <-chan struct{} {
	return v.m.GetStoppingSignal()
}

