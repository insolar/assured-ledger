// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

// E creates an error by the provided description. Description can be a log structure
func E(description interface{}) error {
	return New(description)
}

// EM creates an error by the provided message and description. Description can be a log structure
func EM(msg string, description interface{}) error {
	return NewMsg(msg, description)
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

// RM takes recovered panic, previous error if any, then wraps them together with current stack and message.
// Returns (prevErr) when (recovered) is nil. Description can be a log structure
// NB! Must be called inside defer, e.g. defer func() { err = RM(recover(), err, "msg", x) } ()
func RM(recovered interface{}, prevErr error, msg string, description interface{}) error {
	if recovered == nil {
		return prevErr
	}
	err := WrapPanicExt(recovered, recoverSkipFrames+1)
	d := NewMsg(msg, description)
	return WithDetails(err, WithDetails(prevErr, d))
}

// Rn takes recovered panic and reports it as an error, without a panic mark. Result is wrapped with current stack.
// NB! Must be called inside defer, e.g. defer func() { err = Rn(recover(), err) } ()
func Rn(recovered interface{}, prevErr error) error {
	if recovered == nil {
		return prevErr
	}
	err := WithStackExt(New(recovered), recoverSkipFrames+1)
	return WithDetails(err, prevErr)
}

// Rn takes recovered panic, previous error if any, then wraps them together with current stack and message. But without a panic mark.
// NB! Must be called inside defer, e.g. defer func() { err = RMn(recover(), err, "msg", x) } ()
func RMn(recovered interface{}, prevErr error, msg string, details interface{}) error {
	if recovered == nil {
		return prevErr
	}
	err := WithStackExt(New(recovered), recoverSkipFrames+1)
	d := NewMsg(msg, details)
	return WithDetails(err, WithDetails(prevErr, d))
}
