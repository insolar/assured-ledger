// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"errors"
	stdlog "log"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
)

type AdapterCall struct {
	CallFn   AdapterCallFunc
	Callback *AdapterCallback
	ErrorFn  func(error)
}

var ErrCancelledCall = errors.New("cancelled")

type AdapterCallDelegateFunc func(
	// Not nil
	callFn AdapterCallFunc,
	// Nil for notify calls and when there is no nested call factories are available.
	// Returns false when nested call is impossible (outer call is cancelled or finished)
	nestedCallFn NestedCallFunc,
	// Nil when cancellation is not traced / not configured on SlotMachine adapter / and for notifications
	chainCancel *synckit.ChainedCancel) (AsyncResultFunc, error)

func (c AdapterCall) DelegateAndSendResult(defaultNestedFn CreateFactoryFunc, delegate AdapterCallDelegateFunc) error {
	switch {
	case delegate == nil:
		panic("illegal value")
	case c.Callback == nil:
		return c.delegateNotify(delegate)
	case c.Callback.IsCancelled():
		c.Callback.SendCancel()
		return nil
	}

	result, err := func() (result AsyncResultFunc, err error) {
		defer func() {
			err = RecoverAsyncSlotPanicWithStack("async call", recover(), err)
		}()
		nestedCallFn := c.Callback.getNestedCallHandler(defaultNestedFn)
		return delegate(c.CallFn, nestedCallFn, c.Callback.ChainedCancel())
	}()

	switch {
	case err == nil:
		if !c.Callback.IsCancelled() {
			c.Callback.SendResult(result)
		}
		fallthrough
	case err == ErrCancelledCall:
		c.Callback.SendCancel()
	default:
		c.Callback.SendPanic(err)
	}
	return nil
}

func (c AdapterCall) delegateNotify(delegate AdapterCallDelegateFunc) error {
	result, err := func() (result AsyncResultFunc, err error) {
		defer func() {
			err = RecoverAsyncSlotPanicWithStack("async notify", recover(), err)
		}()
		return delegate(c.CallFn, nil, nil)
	}()
	switch {
	case err == nil:
		if result == nil {
			return nil
		}
		return errors.New("result is unexpected")
	case err == ErrCancelledCall:
		// can't send cancel
		return nil
	default:
		return err
	}
}

func (c AdapterCall) RunAndSendResult(arg interface{}) error {
	return c.DelegateAndSendResult(nil,
		func(callFn AdapterCallFunc, _ NestedCallFunc, _ *synckit.ChainedCancel) (AsyncResultFunc, error) {
			return callFn(arg), nil
		})
}

func (c AdapterCall) ReportError(e error) {
	if c.ErrorFn != nil {
		c.ErrorFn(e)
	} else {
		stdlog.Print(e)
	}
}
