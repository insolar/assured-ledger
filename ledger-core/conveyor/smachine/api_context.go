// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
	"time"
)

type InitFunc func(ctx InitializationContext) StateUpdate
type StateFunc func(ctx ExecutionContext) StateUpdate
type CreateFunc func(ctx ConstructionContext) StateMachine
type MigrateFunc func(ctx MigrationContext) StateUpdate
type FinalizeFunc func(ctx FinalizationContext)
type PostInitFunc func()
type AsyncResultFunc func(ctx AsyncResultContext)
type ErrorHandlerFunc func(ctx FailureContext)
type SubroutineExitFunc func(ctx SubroutineExitContext) StateUpdate
type SubroutineStartFunc func(ctx SubroutineStartContext) InitFunc

type ErrorHandlerAction uint8

const (
	ErrorHandlerDefault ErrorHandlerAction = iota
	ErrorHandlerMute
	ErrorHandlerRecover
	ErrorHandlerRecoverAndWakeUp
	errorHandlerRecoverDenied
)

type BasicContext interface {
	SlotLink() SlotLink
	// can be empty - no parent
	ParentLink() SlotLink
	// not nil
	GetContext() context.Context
}

/*------------------  Contexts for in-order steps -----------------------*/

type DependencyInheritanceMode uint8

const (
	// IgnoreInheritedDependencies allows only local overrides and dependencies provided by SlotMachine to be used for injections.
	// And only factually injected dependencies will be inherited by children.
	IgnoreInheritedDependencies DependencyInheritanceMode = 0

	// for internal use
	copyAllDependencies DependencyInheritanceMode = 1

	// InheritResolvedDependencies makes injection to use resolved dependencies from creator (NB! not from parent)
	// And only immediately injected dependencies will be inherited by children.
	InheritResolvedDependencies DependencyInheritanceMode = 2

	// InheritAllDependencies makes injection to use resolved dependencies from creator (NB! not from parent)
	// And all injected dependencies will be inherited by children.
	InheritAllDependencies DependencyInheritanceMode = 3 // uses copyAllDependencies

	// DiscardResolvedDependencies can be combined with other modes to prevent inheritance by immediate children.
	// This does NOT affect dependencies provided by SlotMachine.
	DiscardResolvedDependencies DependencyInheritanceMode = 4
)

type TerminationCallbackFunc func(result interface{}, err error) AsyncResultFunc

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.ConstructionContext -o ./ -s _mock.go -g
/* During construction SlotLink() will have correct SlotID, but MAY have INVALID status, as slot was not yet created */
type ConstructionContext interface {
	BasicContext

	// OverrideDependency puts a dependency for injector. Value can be nil
	OverrideDependency(id string, v interface{})
	// SetDependencyInheritanceMode sets dependency inheritance mode
	// Precedence of dependencies (from the highest):
	// 1) overrides in this context
	// 2) provided via CreateDefaultValues
	// 3) inherited from creator (when allowed by inheritance mode)
	// 4) slot machine
	// 5) provided to slot machine
	SetDependencyInheritanceMode(DependencyInheritanceMode)

	SetContext(context.Context)
	SetParentLink(SlotLink)

	// SetDefaultTerminationResult sets a default value to be passed to TerminationHandlerFunc when the slot stops.
	SetDefaultTerminationResult(interface{})

	// The given function will be called after SM termination and will have access to SM residual state and to the
	// termination result or error
	SetTerminationCallback(receiver ExecutionContext, getResultFn TerminationCallbackFunc)

	// SetLogTracing sets tracing mode for the slot. Actual impact depends on implementation of a logger.
	SetLogTracing(bool)

	// SetTracerID sets tracer id for the slot. This can't be directly changed after construction. See UpdateDefaultStepLogger()
	SetTracerID(TracerID)
}

type StepLoggerUpdateFunc func(StepLogger, StepLoggerFactoryFunc) StepLogger

type SharedStateContext interface {
	BasicContext

	// Share creates a lazy link to the provided data. Link is invalidated when this SM is stopped.
	// This SM is always has a safe access when active. The shared data is guaranteed to be accessed by only one SM.
	// Access to the data is via ExecutionContext.UseShared().
	// Can be used across different SlotMachines.
	//
	// Do NOT share a reference to a field of SM with ShareDataDirect flag to avoid accidental memory leak.
	// It is recommended to use typed wrappers to access the data.
	Share(data interface{}, flags ShareDataFlags) SharedDataLink
	// Unshare invalidates the given link. Returns true when succeeded.
	// Links created by another slot and links shared with ShareDataUnbound or ShareDataDirect can't be invalidated.
	Unshare(SharedDataLink) bool

	// Publish makes the data to be directly accessible via GetPublished().
	// Data is unpublished when this SM is stopped, except for SharedDataLink with ShareDataUnbound flag.
	// Visibility of key/data is limited by the SlotMachine running this SM.
	//
	// WARNING! There are NO safety guarantees. Publish only immutable data, e.g. publish SharedDataLink.
	// Returns false when key is in use.
	// It is recommended to use typed wrappers to access the data.
	Publish(key, data interface{}) bool
	// Unpublish returns false when key is not in use or the key was published by a different SM.
	// Can always unpublish a key that maps to SharedDataLink with ShareDataUnbound flag.
	Unpublish(key interface{}) bool
	// UnpublishAll removes all keys published by this SM, except for SharedDataLink with ShareDataUnbound.
	UnpublishAll()

	// PublishReplacement updates value for the given key ONLY if key exists and is bound to this slot.
	// When the replacement data is unbound, then the key will also be unbound from this slot.
	// Returns true when key was present and was bound to this slot.
	PublishReplacement(key, data interface{}) bool

	// GetPublished reads data shared by Publish().
	// Visibility of key/data is limited by the SlotMachine running this SM.
	// Returns nil when key is unknown or data is invalidated.
	// It is recommended to use typed wrappers to access the data.
	GetPublished(key interface{}) interface{}
	// GetPublishedLink is a convenience wrapper for GetPublished(). Use SharedDataLink.IsXXX() to check availability.
	// It is recommended to use typed wrappers to access the data.
	GetPublishedLink(key interface{}) SharedDataLink

	// PublishGlobalAlias publishes this Slot globally under the given (key).
	// Published aliases will be unpublished on terminations of SM.
	// Returns false when key is in use.
	PublishGlobalAlias(key interface{}) bool
	// PublishGlobalAliasAndBargeIn publishes this Slot and its barge-in globally under the given (key).
	// Published aliases will be unpublished on terminations of SM.
	// Returns false when key is in use.
	// (handler) can be nil.
	PublishGlobalAliasAndBargeIn(key interface{}, handler BargeInHolder) bool

	// UnpublishGlobalAlias unpublishes the given (key)
	// Returns false when (key) is not published or is published by another slot.
	UnpublishGlobalAlias(key interface{}) bool

	// GetPublishedGlobalAlias reads SlotLink for the given alias (key).
	// When (key) is unknown, then zero/empty SlotLink is returned.
	// Doesn't return handler provided by PublishGlobalAliasAndBargeIn,
	// see GetPublishedGlobalAliasAndBargeIn.
	GetPublishedGlobalAlias(key interface{}) SlotLink

	// GetPublishedGlobalAliasAndBargeIn reads SlotLink and its barge-in for the given alias (key).
	// When (key) is unknown, then zero/empty SlotLink is returned.
	// When barge-in was not set for the (key), then nil holder is returned.
	GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder)
}

/* A context parent for all regular step contexts */
type InOrderStepContext interface {
	SharedStateContext
	SynchronizationContext

	// SetDefaultMigration sets a handler for migrations. Is applied when current SlotStep has no migration handler.
	// MUST be fast as it blocks whole SlotMachine and can't be detached.
	SetDefaultMigration(MigrateFunc)
	// SetDefaultErrorHandler sets a handler for errors and panics. Is applied when current SlotStep has no error handler.
	// MUST be fast as it blocks whole SlotMachine and can't be detached.
	SetDefaultErrorHandler(ErrorHandlerFunc)
	// SetDefaultFlags sets default flags that are merged when SlotStep is set.
	SetDefaultFlags(StepFlags)
	// SetTerminationResult sets a default value to be passed to TerminationHandlerFunc when the slot stops.
	SetTerminationResult(interface{})
	// GetTerminationResult returns a value from the last SetDefaultTerminationResult().
	GetTerminationResult() interface{}

	// SetFinalizer sets a finalization handler. It will be applied on regular stop or on errors, but it won't be called on SlotMachine stop.
	// NB! ErrorHandler is not applied to FinalizeFunc. FinalizeFunc is applied after ErrorHandler.
	SetFinalizer(FinalizeFunc)

	// OverrideDynamicBoost sets boost mark that provides higher priority on scheduling and sync queues. Overrides automatic boost.
	OverrideDynamicBoost(bool)

	// Log returns a slot logger for this context. It is only valid while this context is valid.
	Log() Logger
	// SetLogTracing sets tracing mode for the slot. Actual impact depends on implementation of a logger.
	SetLogTracing(bool)
	// UpdateDefaultStepLogger overrides default step logger. Current logger is provided as argument. Update func can return nil.
	UpdateDefaultStepLogger(StepLoggerUpdateFunc)

	// Jump creates an update to go to the next step. Flags, migrate and error handlers are provided by SetDefaultXXX()
	Jump(StateFunc) StateUpdate
	// JumpExt creates an update to the next step with flags, migrate and error handlers.
	// Flags are merged with SetDefaultFlags() unless StepResetAllFlags is included.
	// Transition must not be nil, other handlers will use SetDefaultXXX() when nil
	JumpExt(SlotStep) StateUpdate
	// RestoreStep is similar to JumpExt, but also can apply sleep state when SlotStep was received from AffectedStep.
	RestoreStep(SlotStep) StateUpdate

	// Error stops SM by calling an error handler.
	Error(error) StateUpdate
	// Stop creates an update to stop the current SM.
	Stop() StateUpdate

	// NewBargeInWithParam creates a barge-in function that can be used to signal or interrupt SM from outside.
	//
	// Provided NewBargeInWithParam sends an async signal to the SM and will be ignored when SM has stopped.
	// When the signal is received by SM the BargeInApplyFunc is invoked. BargeInApplyFunc is safe to access SM.
	// NewBargeInWithParam returns false when SM was stopped at the moment of the call.
	NewBargeInWithParam(BargeInApplyFunc) BargeInWithParam

	// NewBargeIn provides a builder for a simple barge-in.
	NewBargeIn() BargeInBuilder
}

type ShareDataFlags uint32

const (
	// ShareDataWakesUpAfterUse requires that SM called Share() to be woken up after each use of the shared data.
	ShareDataWakesUpAfterUse ShareDataFlags = 1 << iota

	// ShareDataUnbound marks the shared data as immediately accessible and unbound.
	// WARNING! Can ONLY be used for concurrency-safe data. Must NOT keep references to SM or its fields.
	// Data is not bound to SM and will never be invalidated.
	// Keeping such SharedDataLink will retain the data in memory.
	ShareDataUnbound

	// ShareDataDirect marks the shared data as immediately accessible.
	// WARNING! Must NOT keep references to SM or its fields.
	// Data is bound to SM and will be invalidated after stop.
	// But keeping such SharedDataLink will retain the data in memory even when invalidated.
	// ShareDataDirect is overridden by ShareDataUnbound.
	ShareDataDirect

	// ShareDataWithOtherSlotMachines allows access to it form SM's under different SlotMachine(s).
	// By default, access only is allowed to SM's of owner's SlotMachine.
	// This flag is ignored when either ShareDataUnbound or ShareDataDirect is set.
	ShareDataWithOtherSlotMachines
)

type SubroutineCleanupMode uint8

const (
	// All shares and publishes made by this SM will be handed over to the calling SM
	SubroutineCleanupNone SubroutineCleanupMode = 0
	// Shares and non-global aliases made by this SM will be handed over to the calling SM
	SubroutineCleanupGlobalsOnly SubroutineCleanupMode = 1
	// Only shares made by this SM will be handed over to the calling SM
	SubroutineCleanupAnyAliases SubroutineCleanupMode = 2
	// Neither shares nor publishes nor global aliases made by this SM will be handed over to the calling SM
	SubroutineCleanupAliasesAndShares SubroutineCleanupMode = 3
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.InitializationContext -o ./ -s _mock.go -g
type InitializationContext interface {
	InOrderStepContext

	CallBargeInWithParam(b BargeInWithParam, param interface{}) bool
	CallBargeIn(b BargeIn) bool
}

type PostInitStepContext interface {
	InOrderStepContext

	CallSubroutine(SubroutineStateMachine, MigrateFunc, SubroutineExitFunc) StateUpdate
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.ExecutionContext -o ./ -s _mock.go -g
type ExecutionContext interface {
	PostInitStepContext

	StepLink() StepLink
	GetPendingCallCount() int

	// NewBargeInThisStepOnly provides a builder for a simple barge-in. The barge-in function will be ignored if the step has changed.
	// TODO add step-tolerance as for adapter calls
	NewBargeInThisStepOnly() BargeInBuilder

	// InitiateLongRun forces detachment of this slot from SlotMachine's worker to allow slow processing and/or multiple sync calls.
	// WARNING! AVOID this method unless really needed.
	// Can only be called once per step. Detachment remains until end of the step.
	// Detached step will PREVENT access to any bound data shared by this SM.
	// To avoid doubt - detached step, like a normal step, will NOT receive async results, it can only receive result of sync calls.
	//
	// WARNING! SM with a detached step will NOT receive migrations until the detached step is finished.
	// Hence, SM may become inconsistent with other shared objects and injections that could be updated by migrations.
	//
	// Will panic when: (1) not supported by current worker, (2) detachment limit exceeded, (3) called repeatedly.
	InitiateLongRun(LongRunFlags)

	// LogAsync returns slot logger usable in async calls on adapter side. It will be able to report until the slot is invalidated.
	// This logger reports all events as at the step the logger was created at.
	LogAsync() Logger

	// NewChild immediately allocates a new slot and constructs SM. And schedules initialization.
	// It is guaranteed that:
	// 1) the child will start at the same migration state as the creator (caller of this function)
	// 2) initialization of the new slot will happen before any migration
	// 3) the child is created in the same SlotMachine as the creator
	NewChild(CreateFunc) SlotLink
	NewChildExt(CreateFunc, CreateDefaultValues) SlotLink

	// InitChild is same as NewChild, but also grantees that child's initialization will be completed before return.
	// Please prefer NewChild() to avoid unnecessary dependency.
	InitChild(CreateFunc) SlotLink
	// InitChildWithPostInit executes provided callback immediately after child's init step.
	// Inside the callback it is safe to access both the caller and the child.
	InitChildWithPostInit(CreateFunc, PostInitFunc) SlotLink
	InitChildExt(CreateFunc, CreateDefaultValues, PostInitFunc) SlotLink

	// Replace creates an update that after completion of the current step, will stop this SM and will create/start the new SM.
	// The new SM will by default inherit from this SM: parent, context, termination handler/result and injected dependencies.
	// When Replace() is successful, then stopping of this SM will not fire the termination handler.
	// WARNING! Use of SetTerminationHandler() inside CreateFunc will replace the current handler, so it will never fire then.
	Replace(CreateFunc) StateUpdate
	ReplaceExt(CreateFunc, CreateDefaultValues) StateUpdate
	// See Replace()
	ReplaceWith(StateMachine) StateUpdate

	CallBargeInWithParam(b BargeInWithParam, param interface{}) bool
	CallBargeIn(b BargeIn) bool

	// UseShared applies the accessor produced by a SharedDataLink.
	// SharedDataLink can be used across different SlotMachines.
	UseShared(SharedDataAccessor) SharedAccessReport

	// Repeat repeats the current step (it is not considered as change of step).
	// The param limitPerCycle defines how many times this step will be repeated without switching to other slots unless interrupted.
	Repeat(limitPerCycle int) StateUpdate

	// Yield will apply an action chosen by the builder and wait till next work cycle.
	Yield() ConditionalBuilder
	// Poll will apply an action chosen by the builder and wait for a poll interval (configured on SlotMachine).
	Poll() ConditionalBuilder

	// EXPERIMENTAL! SM will apply an action chosen by the builder and wait for activation or stop of the given slot.
	// TODO PLAT-42 WaitActivation(SlotLink) StateConditionalBuilder
	// TODO PLAT-42 WaitStepChange(StepLink, tolerance uint32) StateConditionalBuilder

	// WaitShared will apply an action chosen by the builder and wait for availability of the SharedDataLink.
	WaitShared(SharedDataLink) StateConditionalBuilder
	// WaitAny will apply an action chosen by the builder and wait for any event (even for one irrelevant to this SM).
	WaitAny() ConditionalBuilder
	// WaitAnyUntil will apply an action chosen by the builder and wait for any event (even for one irrelevant to this SM), but not later than the given time.
	WaitAnyUntil(time.Time) StateConditionalBuilder

	// Sleep will apply an action chosen by the builder and wait for an explicit activation of this slot, e.g. any WakeUp() action.
	Sleep() ConditionalBuilder
}

type LongRunFlags uint8

const (
	manualDetach LongRunFlags = 1 << iota
	IgnoreSignal
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.MigrationContext -o ./ -s _mock.go -g
type MigrationContext interface {
	PostInitStepContext

	/* AffectedStep is a step this SM is at during migration */
	AffectedStep() SlotStep

	// SkipMultipleMigrations indicates that multiple pending migrations can be skipped / do not need to be applied individually
	SkipMultipleMigrations()

	/* Stay keeps the last state */
	Stay() StateUpdate
	/* WakeUp makes SM active if it was waiting or polling */
	WakeUp() StateUpdate
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.StateConditionalBuilder -o ./ -s _mock.go -g
type StateConditionalBuilder interface {
	ConditionalBuilder
	// Returns information about the condition being already satisfied
	Decider
	// ThenRepeatOrJump - when the conditional requires wait, then returns Repeat(), otherwise Jump()
	ThenRepeatOrJump(StateFunc) StateUpdate
	// ThenRepeatOrJumpExt - when the conditional requires wait, then returns Repeat(), otherwise JumpExt()
	ThenRepeatOrJumpExt(SlotStep) StateUpdate
}

type CallConditionalBuilder interface {
	ConditionalBuilder
	Sleep() ConditionalBuilder
	Poll() ConditionalBuilder
	WaitAny() ConditionalBuilder
}

type ConditionalBuilder interface {
	ThenJump(StateFunc) StateUpdate
	ThenJumpExt(SlotStep) StateUpdate
	ThenRepeat() StateUpdate
}

/*------------------  Contexts for out-of-order steps -----------------------*/

type AsyncResultContext interface {
	BasicContext

	// Returns a slot logger for this context. It is only valid while this context is valid.
	Log() Logger

	// Makes SM active if it was waiting or polling
	WakeUp()
}

type BargeInApplyFunc func(interface{}) BargeInCallbackFunc
type BargeInCallbackFunc func(ctx BargeInContext) StateUpdate

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.BargeInBuilder -o ./ -s _mock.go -g
type BargeInBuilder interface {
	// WithJumpExt creates a BargeIn that will change SM's step and wake it up
	WithJumpExt(SlotStep) BargeIn
	// WithJump creates a BargeIn that will change SM's step and wake it up
	WithJump(StateFunc) BargeIn
	// WithWakeUp creates a BargeIn that will wake up SM at its current step
	WithWakeUp() BargeIn
	// WithStop creates a BargeIn that will stop SM
	WithStop() BargeIn
	// WithError creates a BargeIn that will stop SM with the given error
	WithError(error) BargeIn
}

type interruptContext interface {
	BasicContext

	// Log returns a slot logger for this context. It is only valid while this context is valid.
	Log() Logger

	AffectedStep() SlotStep

	JumpExt(SlotStep) StateUpdate
	Jump(StateFunc) StateUpdate
	RestoreStep(SlotStep) StateUpdate

	// Error will stop SM by calling an error handler.
	Error(error) StateUpdate

	Stop() StateUpdate
}

type BargeInContext interface {
	interruptContext

	// IsAtOriginalStep returns true when SM step wasn't change since barge-in creation
	IsAtOriginalStep() bool

	// Stay keeps the last state
	Stay() StateUpdate
	// Makes SM active if it was waiting or polling
	WakeUp() StateUpdate
}

type SubroutineExitContext interface {
	interruptContext

	EventParam() interface{}

	// GetError returns an error when subroutine was stopped by an error
	GetError() error
}

type SubroutineStartContext interface {
	BasicContext

	// Log returns a slot logger for this context. It is only valid while this context is valid.
	Log() Logger

	SetSubroutineCleanupMode(SubroutineCleanupMode)
}

var _ LimitedExecutionContext = ExecutionContext(nil) // MUST be assignable
type LimitedExecutionContext interface {
	SharedStateContext
	minimalSynchronizationContext

	// NewChild - See ExecutionContext.NewChild
	NewChild(CreateFunc) SlotLink
	// InitChild - See ExecutionContext.InitChild
	InitChild(CreateFunc) SlotLink
}

type FinalizationContext interface {
	LimitedExecutionContext

	// GetTerminationResult returns a last value set by SetTerminationResult()
	GetTerminationResult() interface{}

	// SetTerminationResult sets a value to be passed to TerminationHandlerFunc.
	SetTerminationResult(interface{})

	// GetError returns an error when finalization was caused by an error.
	GetError() error
}

type FailureContext interface {
	FinalizationContext

	// AffectedStep is a step the slot is at
	AffectedStep() SlotStep

	// GetError returns a reason of the failure
	GetError() error

	// IsPanic is false when the error was initiated by ctx.Error(). When true, then GetError() should be SlotPanicError
	IsPanic() bool

	// GetArea provides information about area type where the error/panic has appeared
	GetArea() SlotPanicArea

	// CanRecover is true when this error can be recovered by SetAction(ErrorHandlerRecover).
	// A panic inside async call / callback can be recovered.
	CanRecover() bool

	// SetAction chooses an action to be applied.
	// Recovery actions will be ignored when CanRecover() is false.
	SetAction(ErrorHandlerAction)

	// UnsetFinalizer is an equivalent InOrderStepContext.SetFinalizer(nil).
	UnsetFinalizer()
}
