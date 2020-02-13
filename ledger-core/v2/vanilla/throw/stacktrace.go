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

import (
	"bytes"
	"io"
	"runtime"
	"runtime/debug"
)

const StackTracePrefix = "Stack trace: "

type StackTrace interface {
	StackTraceAsText() string
	WriteStackTraceTo(writer io.Writer) error
}

func CaptureStack(skipFrames int) StackTrace {
	return byteStackTrace(captureStack(skipFrames, false))
}

func CaptureStackTop(skipFrames int) StackTrace {
	return byteStackTrace(captureStack(skipFrames, true))
}

func equalStackTrace(st0, st1 StackTrace) bool {
	if bst0, ok := st0.(byteStackTrace); ok {
		if bst1, ok := st1.(byteStackTrace); ok {
			return bytes.Equal(bst0, bst1)
		}
	}
	return false
}

func IsInSystemPanic(skipFrames int) bool {
	pc := make([]uintptr, 1)
	if runtime.Callers(skipFrames+2, pc) != 1 {
		return false
	}
	n := runtime.FuncForPC(pc[0]).Name()
	return n == "runtime.preprintpanics"
}

type byteStackTrace []byte

func (v byteStackTrace) WriteStackTraceTo(w io.Writer) error {
	_, err := w.Write(v)
	return err
}

func (v byteStackTrace) StackTraceAsText() string {
	return string(v)
}

func (v byteStackTrace) LogString() string {
	return StackTracePrefix + string(v)
}

func captureStack(skipFrames int, limitFrames bool) []byte {
	stackBytes := debug.Stack()
	capacity := cap(stackBytes)
	if i := bytes.IndexByte(stackBytes, '\n'); i > 0 {
		stackBytes = stackBytes[i+1:]
	} else {
		// strange result, let be safe
		return stackBytes
	}

	if i := indexOfFrame(stackBytes, skipFrames+2); i > 0 {
		stackBytes = stackBytes[i:]

		if limitFrames {
			if i := indexOfFrame(stackBytes, 1); i > 0 {
				stackBytes = stackBytes[:i]
			}
		}
	}

	stackBytes = bytes.TrimSpace(stackBytes)

	if capacity > len(stackBytes)<<1 {
		return append(make([]byte, 0, len(stackBytes)), stackBytes...)
	}

	return stackBytes
}

var frameSep = []byte("\n\t")

func indexOfFrame(stackBytes []byte, skipFrames int) int {
	offset := 0
	for ; skipFrames > 0; skipFrames-- {
		if i := bytes.Index(stackBytes[offset:], frameSep); i > 0 {
			offset += i + len(frameSep)
			if j := bytes.IndexByte(stackBytes[offset:], '\n'); j > 0 {
				offset += j + 1
				continue
			}
		}
		return -1
	}
	return offset
}
