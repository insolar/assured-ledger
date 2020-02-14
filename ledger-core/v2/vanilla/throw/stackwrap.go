//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

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
	return stackWrap{st: CaptureStack(skipFrames + 1), err: err}
}

// WithStack wraps the error with stack's topmost entry after skipping the given number of frames. Nil value will return nil.
func WithStackTopExt(err error, skipFrames int) error {
	if err == nil {
		return nil
	}
	if skipFrames < 0 {
		skipFrames = 0
	}
	return stackWrap{st: CaptureStackTop(skipFrames + 1), err: err}
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
	}

	return detailsWrap{err: predecessor, details: wrap(d),
		isComparable: reflect.TypeOf(details).Comparable()}
}

func WithStackAndDetails(predecessor error, details interface{}) error {
	return WithStack(WithDetails(predecessor, details))
}
