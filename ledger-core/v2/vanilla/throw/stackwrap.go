// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import "reflect"

func WithStack(err error) error {
	return WithStackExt(err, 1)
}

// WithStackTop wraps the error with stack's topmost entry only. Nil value will return nil.
// Use this method to augment an error specific to a code location.
func WithStackTop(err error) error {
	return WithStackTopExt(err, 1)
}

// WithStack wraps the error with stack with the given number of frames skipped. Nil value will return nil.
func WithStackExt(err error, skipFrames int) error {
	if err == nil {
		return nil
	}
	if skipFrames < 0 {
		skipFrames = 0
	}
	return withStack(err, CaptureStack(skipFrames+1))
}

// WithStack wraps the error with stack's topmost entry after skipping the given number of frames. Nil value will return nil.
func WithStackTopExt(err error, skipFrames int) error {
	if err == nil {
		return nil
	}
	if skipFrames < 0 {
		skipFrames = 0
	}
	return withStack(err, CaptureStackTop(skipFrames+1))
}

func reuseSupersetTrace(current, wrapped StackTrace) StackTrace {
	switch {
	case current == nil:
		return wrapped
	case wrapped == nil:
		return current
	}
	switch CompareStackTraceExt(current, wrapped, SameMethod) {
	case SubsetTrace, TopTrace:
		return wrapped
	default:
		return current
	}
}

func withStack(err error, st StackTrace) stackWrap {
	if sth := OutermostStack(err); sth != nil {
		return stackWrap{st: st, stDeepest: reuseSupersetTrace(st, sth.DeepestStackTrace()), err: err}
	}
	return stackWrap{st: st, err: err}
}

func WithDetails(predecessor error, details interface{}) error {
	switch {
	case details == nil:
		return predecessor
	case predecessor == nil:
		return Wrap(details)
	default:
		return withDetails(predecessor, details)
	}
}

func withDetails(predecessor error, details interface{}) error {
	var d fmtWrap
	switch vv := details.(type) {
	case fmtWrap:
		d = vv
	case panicWrap:
		d = vv.fmtWrap
	case msgWrap:
		d.msg = vv.msg
	case stackWrap:
		if vv.err == nil {
			return predecessor
		}
		return withDetails(predecessor, vv.err)
	case nil:
		// nil is handled by caller
		panic("illegal value")
	default:
		d = wrapInternal(details)
	}

	return detailsWrap{err: predecessor, details: d, isComparable: reflect.TypeOf(details).Comparable()}
}

func WithStackAndDetails(predecessor error, details interface{}) error {
	return WithStackExt(WithDetails(predecessor, details), 1)
}
