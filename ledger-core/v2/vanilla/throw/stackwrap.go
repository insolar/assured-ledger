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

func WithStack(err error) error {
	return WithStackEx(err, 1)
}

func WithStackEx(err error, skipFrames int) error {
	if err == nil {
		return nil
	}
	if skipFrames < 0 {
		skipFrames = 0
	}
	return stackWrap{st: CaptureStack(skipFrames + 1), err: err}
}

type stackWrap struct {
	_logignore struct{} // will be ignored by struct-logger
	st         StackTrace
	err        error
}

func (v stackWrap) StackTrace() StackTrace {
	return v.st
}

func (v stackWrap) Reason() error {
	return v.Unwrap()
}

func (v stackWrap) Unwrap() error {
	return v.err
}

func (v stackWrap) LogString() string {
	if vv, ok := v.err.(logStringer); ok {
		return vv.LogString()
	}

	return v.err.Error()
}

func (v stackWrap) Error() string {
	if v.st == nil {
		return v.err.Error()
	}
	return v.err.Error() + "\n" + StackTracePrefix + v.st.StackTraceAsText()
}
