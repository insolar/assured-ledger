package smachine

import (
	"context"
)

type AdapterID string

func (v AdapterID) IsEmpty() bool {
	return len(v) == 0
}

type AdapterCallFunc func(ctx context.Context, arg interface{}) AsyncResultFunc
type AdapterNotifyFunc func(ctx context.Context, arg interface{})
type CreateFactoryFunc func(eventPayload interface{}) CreateFunc

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.AsyncCallRequester -o ./ -s _mock.go -g
type AsyncCallRequester interface {
	// WithCancel allocates and provides cancellation function. Repeated call returns same.
	WithCancel(*context.CancelFunc) AsyncCallRequester
	// WithNested sets a handler to map nested calls from the target adapter to new SMs
	// If this handler is nil or returns nil, then a default handler of the adapter will be in use.
	// To block a nested event - return non-nil CreateFunc, and then return nil from CreateFunc.
	WithNested(CreateFactoryFunc) AsyncCallRequester
	// WithFlags sets AsyncCallFlags. AutoWakeUp is set by default.
	WithFlags(flags AsyncCallFlags) AsyncCallRequester
	// WithoutAutoWakeUp removes AutoWakeUp. See AsyncCallFlags.
	WithoutAutoWakeUp() AsyncCallRequester
	// WithLog sets internal logging for the call and its result.
	// This mode is set automatically when tracing is active or StepElevatedLog is set.
	WithLog(isLogging bool) AsyncCallRequester

	// Start starts this async call
	Start()
	// DelayedStart creates an update that can be returned as a new state and will ONLY be executed if returned as a new state
	DelayedStart() CallConditionalBuilder
}

type AsyncCallFlags uint8

const (
	/*
		CallBoundToStep indicates that a call stays valid for this step (where the call is made) and for a next step.
		When SM will went further, the call or its result will be cancelled / ignored.
		NB! This cancel functionality is PASSIVE, an adapter should check this status explicitly.
	*/
	CallBoundToStep AsyncCallFlags = 1 << iota

	// WakeUpBoundToStep indicates if a wakeup from call's result will be valid for this step (where the call is made) and for a next step.
	WakeUpBoundToStep
	// WakeUpOnResult enables receiving of call's successful result to wake up the slot without WakeUp(). Affected by WakeUpBoundToStep.
	WakeUpOnResult
	// WakeUpOnResult enables a caller to be woken up when the async request was cancelled by an adapter. Affected by WakeUpBoundToStep.
	WakeUpOnCancel
)

const AutoWakeUp = WakeUpOnResult | WakeUpOnCancel

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.NotifyRequester -o ./ -s _mock.go -g
type NotifyRequester interface {
	// WithLog sets internal logging for the call. This mode is set automatically when tracing is active or StepElevatedLog is set.
	WithLog(isLogging bool) NotifyRequester
	// Send immediately sends notification
	Send()
	// DelayedSend creates an update that can be returned as a new state and will ONLY be executed if returned as a new state
	DelayedSend() CallConditionalBuilder
	// DelegationFunc returns a function that can be called outside of SM to send a notification.
	SendFunc() func()
}

type SyncCallRequester interface {
	// WithNested sets a handler to map nested calls from the target adapter to new SMs.
	// See AsyncCallRequester.WithNested() for details.
	WithNested(CreateFactoryFunc) SyncCallRequester
	// WithLog sets internal logging for the call. This mode is set automatically when tracing is active or StepElevatedLog is set.
	WithLog(isLogging bool) SyncCallRequester

	// TryCall returns true when the call was successful, or false if cancelled. May return false on a signal - depends on context mode.
	TryCall() bool
	// Call may panic on migrate - depends on context mode
	Call()
}

// Provides execution of calls to an adapter.
type AdapterExecutor interface {
	// StartCall schedules asynchronous execution, MAY return native cancellation function, but only if supported.
	// This method MUST be fast and MUST NOT lock up. May panic on queue overflow.
	StartCall(ctx context.Context, fn AdapterCallFunc, callback *AdapterCallback, needCancel bool) context.CancelFunc

	// SendNotify schedules asynchronous, fire-and-forget execution.
	// This method MUST be fast and MUST NOT lock up. May panic on queue overflow.
	SendNotify(context.Context, AdapterNotifyFunc)

	// TODO PLAT-41 pass in a cancellation object
	// TrySyncCall performs sync call ONLY if *natively* supported by the adapter, otherwise must return (false, nil)
	TrySyncCall(context.Context, AdapterCallFunc) (bool, AsyncResultFunc)
}
