// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

// IllegalValue is to indicate that an argument provided to a calling function is incorrect
// This error captures the topmost entry of caller's stack.
func IllegalValue() error {
	return newMsg("illegal value", 0)
}

// IllegalState is to indicate that an internal state of a function/object is incorrect or unexpected
// This error captures the topmost entry of caller's stack.
func IllegalState() error {
	return newMsg("illegal state", 0)
}

// Unsupported is to indicate that a calling function is unsupported intentionally and will remain so for awhile
// This error captures the topmost entry of caller's stack.
func Unsupported() error {
	return newMsg("unsupported", 0)
}

// NotImplemented is to indicate that a calling function was not yet implemented, but it is expected to be completed soon
// This error captures the topmost entry of caller's stack.
func NotImplemented() error {
	return newMsg("not implemented", 0)
}

// Impossible is to indicate that the point can never be reached.
// This error captures the topmost entry of caller's stack.
func Impossible() error {
	return newMsg("impossible", 0)
}

// FailHere creates an error that captures the topmost entry of caller's stack.
func FailHere(msg string) error {
	return newMsg(msg, 0)
}

// FailCaller creates an error that captures the entry of caller's stack after skipping (skipFrames).
func FailCaller(msg string, skipFrames int) error {
	return newMsg(msg, skipFrames)
}

func newMsg(msg string, skipFrames int) msgWrap {
	return msgWrap{st: CaptureStackTop(skipFrames + 2), msg: msg}
}
