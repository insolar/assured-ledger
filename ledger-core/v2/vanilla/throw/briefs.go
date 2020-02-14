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
	err := WrapPanicExt(recovered, recoverSkipFrames+1)
	d := WrapMsg(msg, details)
	return WithDetails(err, WithDetails(prevErr, d))
}
