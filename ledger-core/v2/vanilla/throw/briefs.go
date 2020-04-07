// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

// E creates an error by the provided message and description. Description can be a log structure
func E(msg string, description ...interface{}) error {
	if len(description) == 1 {
		if err, ok := description[0].(error); ok {
			return W(err, msg)
		}
	}
	return New(msg, description...)
}

// W wraps the given error with provided message and details.
// Returns nil when (err) == nil
func W(err error, msg string, description ...interface{}) error {
	return WithDetails(err, New(msg, description...))
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

// RW takes recovered panic, previous error if any, then wraps them together with current stack and message.
// Returns (prevErr) when (recovered) is nil. Description can be a log structure
// NB! Must be called inside defer, e.g. defer func() { err = RW(recover(), err, "msg", x) } ()
func RW(recovered interface{}, prevErr error, msg string, description ...interface{}) error {
	if recovered == nil {
		return prevErr
	}
	err := WrapPanicExt(recovered, recoverSkipFrames+1)
	d := New(msg, description...)
	return WithDetails(err, WithDetails(prevErr, d))
}

// Rn takes recovered panic and reports it as an error, without a panic mark. Result is wrapped with current stack.
// NB! Must be called inside defer, e.g. defer func() { err = Rn(recover(), err) } ()
func Rn(recovered interface{}, prevErr error) error {
	if recovered == nil {
		return prevErr
	}
	err := WithStackExt(NewDescription(recovered), recoverSkipFrames+1)
	return WithDetails(err, prevErr)
}

// Rn takes recovered panic, previous error if any, then wraps them together with current stack and message. But without a panic mark.
// NB! Must be called inside defer, e.g. defer func() { err = RWn(recover(), err, "msg", x) } ()
func RWn(recovered interface{}, prevErr error, msg string, details ...interface{}) error {
	if recovered == nil {
		return prevErr
	}
	err := WithStackExt(NewDescription(recovered), recoverSkipFrames+1)
	d := New(msg, details...)
	return WithDetails(err, WithDetails(prevErr, d))
}
