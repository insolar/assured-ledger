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

func (c AdapterCall) DelegateAndSendResult(defaultNestedFn CreateFactoryFunc, delegate AdapterCallDelegateFunc) (func(), error) {
	switch {
	case delegate == nil:
		panic("illegal value")
	case c.Callback == nil:
		return nil, c.delegateNotify(delegate)
	case c.Callback.IsCancelled():
		c.Callback.SendCancel()
		return nil, nil
	}

	result, err := func() (result AsyncResultFunc, err error) {
		defer func() {
			err = RecoverSlotPanicWithStack("async call", recover(), err, AsyncCallArea)
		}()
		nestedCallFn := c.Callback.getNestedCallHandler(defaultNestedFn)
		return delegate(c.CallFn, nestedCallFn, c.Callback.ChainedCancel())
	}()

	switch {
	case err == nil:
		return func() {
			if !c.Callback.IsCancelled() {
				c.Callback.SendResult(result)
			}
			c.Callback.SendCancel()
		}, nil
	case err == ErrCancelledCall:
		return c.Callback.SendCancel, nil
	default:
		return func() {
			c.Callback.SendPanic(err)
		}, nil
	}
}

func (c AdapterCall) delegateNotify(delegate AdapterCallDelegateFunc) error {
	result, err := func() (result AsyncResultFunc, err error) {
		defer func() {
			err = RecoverSlotPanicWithStack("async notify", recover(), err, AsyncCallArea)
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
	fn, err := c.DelegateAndSendResult(nil,
		func(callFn AdapterCallFunc, _ NestedCallFunc, _ *synckit.ChainedCancel) (AsyncResultFunc, error) {
			return callFn(arg), nil
		})
	if fn != nil {
		fn()
	}
	return err
}

func (c AdapterCall) ReportError(e error) {
	if c.ErrorFn != nil {
		c.ErrorFn(e)
	} else {
		stdlog.Print(e)
	}
}
