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

// IllegalValue is to indicate that an argument provided to a calling function is incorrect
// This error captures the topmost entry of caller's stack.
func IllegalValue() error {
	return newMsg("illegal value")
}

// IllegalState is to indicate that an internal state of a function/object is incorrect or unexpected
// This error captures the topmost entry of caller's stack.
func IllegalState() error {
	return newMsg("illegal state")
}

// Unsupported is to indicate that a calling function is unsupported intentionally and will remain so for awhile
// This error captures the topmost entry of caller's stack.
func Unsupported() error {
	return newMsg("unsupported")
}

// NotImplemented is to indicate that a calling function was not yet implemented, but it is expected to be completed soon
// This error captures the topmost entry of caller's stack.
func NotImplemented() error {
	return newMsg("not implemented")
}

func newMsg(msg string) msgWrap {
	return msgWrap{st: CaptureStackTop(2), msg: msg}
}

type msgWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	msg        string
}

func (v msgWrap) StackTrace() StackTrace {
	return v.st
}

func (v msgWrap) Error() string {
	return v.msg
}

func (v msgWrap) String() string {
	if v.st == nil {
		return v.msg
	}
	return v.msg + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}
