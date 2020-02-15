// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import "fmt"

// E creates an error by the provided description
func E(description interface{}) error {
	return Wrap(description)
}

// EM creates an error by the provided message and description
func EM(msg string, description interface{}) error {
	return WrapMsg(msg, description)
}

// R takes recovered panic and previous error if any, then wraps them together with current stack
// Returns (prevErr) when (recovered) is nil.
// NB! Must be called inside defer, e.g. defer func() { err = R(recover(), err) } ()
func R(recovered interface{}, prevErr error) error {
	if recovered == nil {
		return prevErr
	}
	err := WrapPanicExt(recovered, recoverSkipFrames+1)
	if prevErr == nil {
		return err
	}
	return WithDetails(err, prevErr)
}

// RM takes recovered panic, previous error if any, then wraps them together with current stack and message details.
// Returns (prevErr) when (recovered) is nil.
// NB! Must be called inside defer, e.g. defer func() { err = RM(recover(), err, "msg", x) } ()
func RM(recovered interface{}, prevErr error, msg string, details interface{}) error {
	if recovered == nil {
		return prevErr
	}
	fmt.Println("Re>>>> ", recovered)
	err := WrapPanicExt(recovered, recoverSkipFrames+1)
	fmt.Println("RM>>>> ", err)
	d := WrapMsg(msg, details)
	return WithDetails(err, WithDetails(prevErr, d))
}
