// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
)

type AdapterId string

func (v AdapterId) IsEmpty() bool {
	return len(v) == 0
}

type AdapterCallFunc func(arg interface{}) AsyncResultFunc
type AdapterNotifyFunc func(arg interface{})
type CreateFactoryFunc func(eventPayload interface{}) CreateFunc

type AsyncCallRequester interface {
	// Allocates and provides cancellation function. Repeated call returns same.
	WithCancel(*context.CancelFunc) AsyncCallRequester
	// Sets a handler to map nested calls from the target adapter to new SMs
	// If this handler is nil or returns nil, then a default handler of the adapter will be in use.
	// To block a nested event - return non-nil CreateFunc, and then return nil from CreateFunc.
	WithNested(CreateFactoryFunc) AsyncCallRequester
	// See AsyncCallFlags, set to AutoWakeUp by default.
	WithFlags(flags AsyncCallFlags) AsyncCallRequester
	// Sets internal logging for the call and its result.
	// This mode is set automatically when tracing is active or StepElevatedLog is set.
	WithLog(isLogging bool) AsyncCallRequester

	// Starts async call
	Start()
	// Creates an update that can be returned as a new state and will ONLY be executed if returned as a new state
	DelayedStart() CallConditionalBuilder
}

type AsyncCallFlags uint8

const (
	/*
		Call stays valid for this step (where the call is made) and for a next step.
		When SM will went further, the call or its result will be cancelled / ignored.
		NB! This cancel functionality is PASSIVE, an adapter should check this status explicitly.
	*/
	CallBoundToStep AsyncCallFlags = iota << 1

	// When set, a wakeup from call's result will be valid for this step (where the call is made) and for a next step.
	WakeUpBoundToStep
	//	When set, receiving of call's successful result will wake up the slot without WakeUp(). Affected by WakeUpBoundToStep.
	WakeUpOnResult
	// Caller will be woken up when the async request was cancelled by an adapter. Affected by WakeUpBoundToStep.
	WakeUpOnCancel
)

const AutoWakeUp = WakeUpOnResult | WakeUpOnCancel

type NotifyRequester interface {
	// Sets internal logging for the call. This mode is set automatically when tracing is active or StepElevatedLog is set.
	WithLog(isLogging bool) NotifyRequester
	// Sends notify
	Send()
	// Creates an update that can be returned as a new state and will ONLY be executed if returned as a new state
	DelayedSend() CallConditionalBuilder
}

type SyncCallRequester interface {
	// Sets a handler to map nested calls from the target adapter to new SMs.
	// See AsyncCallRequester.WithNested() for details.
	WithNested(CreateFactoryFunc) SyncCallRequester
	// Sets internal logging for the call. This mode is set automatically when tracing is active or StepElevatedLog is set.
	WithLog(isLogging bool) SyncCallRequester

	// Returns true when the call was successful, or false if cancelled. May return false on a signal - depends on context mode.
	TryCall() bool
	// May panic on migrate - depends on context mode
	Call()
}

// Provides execution of calls to an adapter.
type AdapterExecutor interface {
	// Schedules asynchronous execution, MAY return native cancellation function, but only if supported.
	// This method MUST be fast and MUST NOT lock up. May panic on queue overflow.
	StartCall(fn AdapterCallFunc, callback *AdapterCallback, needCancel bool) context.CancelFunc

	// Schedules asynchronous, fire-and-forget execution.
	// This method MUST be fast and MUST NOT lock up. May panic on queue overflow.
	SendNotify(AdapterNotifyFunc)

	// Performs sync call ONLY if *natively* supported by the adapter, otherwise must return (false, nil)
	// TODO PLAT-41 pass in a cancellation object
	TrySyncCall(AdapterCallFunc) (bool, AsyncResultFunc)
}
