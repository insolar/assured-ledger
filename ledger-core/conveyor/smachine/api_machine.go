package smachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

// SlotMachineHolder is a set of safe for async call methods of SlotMachine
type SlotMachineHolder interface {
	GetMachineID() string

	injector.DependencyContainer
	AddDependency(v interface{})
	AddInterfaceDependency(v interface{})
	GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder)

	AddNew(context.Context, StateMachine, CreateDefaultValues) (SlotLink, bool)
	AddNewByFunc(context.Context, CreateFunc, CreateDefaultValues) (SlotLink, bool)

	OccupiedSlotCount() int
	AllocatedSlotCount() int

	ScheduleCall(fn MachineCallFunc, isSignal bool) bool

	Stop() bool
	GetStoppingSignal() <-chan struct{}
}

type MachineCallFunc func(MachineCallContext)

// Provides easy-to-use access to functions of the SlotMachine that require a proper worker / concurrency control
type MachineCallContext interface {
	SlotMachine() *SlotMachine
	GetMachineID() string

	AddNew(context.Context, StateMachine, CreateDefaultValues) (SlotLink, bool)
	AddNewByFunc(context.Context, CreateFunc, CreateDefaultValues) (SlotLink, bool)

	// CallDirectBargeIn executes (fn) like it was provided by the given slot (link).
	// WARNING! USE with extreme caution, as it WILL interfere with normal SM behavior, including initialization.
	CallDirectBargeIn(link StepLink, fn BargeInCallbackFunc) bool

	CallBargeInWithParam(b BargeInWithParam, param interface{}) bool
	CallBargeIn(b BargeIn) bool

	GetPublished(key interface{}) interface{}
	GetPublishedLink(key interface{}) SharedDataLink

	GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder)

	Migrate(beforeFn func())
	Cleanup()
	Stop()

	// See SynchronizationContext
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
