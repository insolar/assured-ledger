// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import "context"

type MachineCallFunc func(MachineCallContext)

// Provides easy-to-use access to functions of the SlotMachine that require a proper worker / concurrency control
type MachineCallContext interface {
	SlotMachine() *SlotMachine
	GetMachineId() string

	AddNew(context.Context, StateMachine, CreateDefaultValues) SlotLink
	AddNewByFunc(context.Context, CreateFunc, CreateDefaultValues) (SlotLink, bool)

	CallDirectBargeIn(link StepLink, fn BargeInCallbackFunc) bool
	CallBargeInWithParam(b BargeInWithParam, param interface{}) bool
	CallBargeIn(b BargeIn) bool

	GetPublished(key interface{}) interface{}
	GetPublishedLink(key interface{}) SharedDataLink

	GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder)

	Migrate(beforeFn func())
	Cleanup()
	Stop()

	//See SynchronizationContext
	ApplyAdjustment(SyncAdjustment) bool
	Check(SyncLink) BoolDecision
}

func ScheduleCallTo(link SlotLink, fn MachineCallFunc, isSignal bool) bool {
	if fn == nil {
		panic("illegal value")
	}
	m := link.getActiveMachine()
	if m == nil {
		return false
	}
	return m.ScheduleCall(fn, isSignal)
}
