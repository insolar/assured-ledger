// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

type AdapterCallbackFunc func(isValid bool, fn AsyncResultFunc, err error) bool

func NewAdapterCallback(adapterID AdapterID, caller StepLink, callbackOverride AdapterCallbackFunc, flags AsyncCallFlags,
	nestedFn CreateFactoryFunc) *AdapterCallback {
	return &AdapterCallback{
		adapterID:  adapterID,
		caller:     caller,
		callbackFn: callbackOverride,
		cancel:     nil,
		nestedFn:   nestedFn,
		state:      0,
		flags:      flags,
	}
}

type AdapterCallback struct {
	adapterID  AdapterID
	caller     StepLink
	callbackFn AdapterCallbackFunc
	cancel     *synckit.ChainedCancel
	nestedFn   CreateFactoryFunc
	state      uint32 // atomic
	flags      AsyncCallFlags
}

func (c *AdapterCallback) Prepare(needCancel bool) context.CancelFunc {
	if !atomic.CompareAndSwapUint32(&c.state, 0, 1) {
		panic("illegal state - in use")
	}
	if c.cancel != nil {
		panic("illegal state")
	}
	if !needCancel {
		return nil
	}

	c.cancel = synckit.NewChainedCancel()
	return c.cancel.Cancel
}

const stepBondTolerance uint32 = 1

func (c *AdapterCallback) canCall() bool {
	return c.flags&CallBoundToStep == 0 || c.caller.IsNearStep(stepBondTolerance)
}

func (c *AdapterCallback) IsCancelled() bool {
	return c.cancel.IsCancelled() || !c.canCall()
}

func (c *AdapterCallback) ChainedCancel() *synckit.ChainedCancel {
	return c.cancel
}

func (c *AdapterCallback) SendResult(result AsyncResultFunc) {
	if result == nil {
		// NB! Do NOT ignore "result = nil" - callback need to decrement async call count
		result = func(ctx AsyncResultContext) {}
	}
	c.callback(false, result, nil)
}

func (c *AdapterCallback) SendPanic(err error) {
	if err == nil {
		panic("illegal value")
	}
	c.callback(false, nil, err)
}

// can be called repeatedly
func (c *AdapterCallback) SendCancel() {
	c.cancel.Cancel()
	c.callback(true, nil, nil)
}

func (c *AdapterCallback) callback(isCancel bool, resultFn AsyncResultFunc, err error) {
	switch atomic.SwapUint32(&c.state, 2) {
	case 2:
		if isCancel {
			// repeated cancel are safe
			return
		}
		panic("illegal state - repeated callback")
	case 0:
		panic("illegal state - unprepared")
	}

	switch {
	case !c.canCall():
		if c.callbackFn != nil {
			c.callbackFn(false, resultFn, err)
		}
		return
	case c.callbackFn != nil:
		if c.callbackFn(true, resultFn, err) {
			return
		}
	}

	c.caller.s.getMachine().queueAsyncResultCallback(c.caller, c.flags, c.cancel, resultFn, err, false)
}

func (m *SlotMachine) queueAsyncResultCallback(callerLink StepLink, flags AsyncCallFlags, cancel *synckit.ChainedCancel,
	resultFn AsyncResultFunc, err error, ignoreAsyncCount bool,
) bool { // nolint:unparam
	if m == nil {
		return false
	}

	return m.queueAsyncCallback(callerLink.SlotLink, func(slot *Slot, worker DetachableSlotWorker, err error) StateUpdate {
		if !ignoreAsyncCount {
			slot.decAsyncCount()
		}

		wakeupAllowed := flags&WakeUpBoundToStep == 0 || callerLink.IsNearStep(stepBondTolerance)
		wakeup := false
		switch {
		case err != nil:
			return StateUpdate{} // result will be replaced by queueAsyncCallback()

		case resultFn == nil /* cancelled by adapter */ || cancel.IsCancelled():
			wakeup = flags&WakeUpOnCancel != 0

		default:
			rc := asyncResultContext{s: slot, w: worker}
			wakeupResult := rc.executeResult(resultFn)
			wakeup = wakeupResult || flags&WakeUpOnResult != 0
		}

		if wakeup && wakeupAllowed {
			return newStateUpdateTemplate(updCtxAsyncCallback, 0, stateUpdWakeup).newNoArg()
		}
		return newStateUpdateTemplate(updCtxAsyncCallback, 0, stateUpdNoChange).newNoArg()
	}, err)
}

func (c *AdapterCallback) SendNested(defaultFactoryFn CreateFactoryFunc, payload interface{}) error {

	m := c.caller.getActiveMachine()
	if m == nil {
		return fmt.Errorf("target SlotMachine is stopping/stopped")
	}

	createFn := func(factoryFn CreateFactoryFunc) (bool, error) {
		if factoryFn == nil {
			return false, nil
		}
		if cf := factoryFn(payload); cf != nil {
			switch link, ok := m.AddNested(c.adapterID, c.caller.SlotLink, cf); {
			case ok:
				return true, nil
			case link.IsZero():
				return true, fmt.Errorf("target SlotMachine is stopping/stopped")
			default:
				return true, fmt.Errorf("cancelled by constructor")
			}
		}
		return false, nil
	}

	if ok, err := createFn(c.nestedFn); ok {
		return err
	}
	if ok, err := createFn(defaultFactoryFn); ok {
		return err
	}
	return fmt.Errorf("unknown payload for nested call")
}

type NestedCallFunc func(interface{}) error

func (c *AdapterCallback) getNestedCallHandler(defFactoryFn CreateFactoryFunc) NestedCallFunc {
	if defFactoryFn == nil && c.nestedFn == nil {
		return nil
	}
	return func(v interface{}) error {
		return c.SendNested(defFactoryFn, v)
	}
}
