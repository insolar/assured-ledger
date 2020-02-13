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

// FailsHere creates an error that captures the topmost entry of caller's stack.
func FailsHere(msg string, skipFrames int) error {
	return newMsg(msg, skipFrames)
}

func newMsg(msg string, skipFrames int) msgWrap {
	return msgWrap{st: CaptureStackTop(skipFrames + 2), msg: msg}
}
