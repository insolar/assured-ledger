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
	"strconv"
)

const StackTracePrefix = "Stack trace: "

type StackTrace interface {
	StackTraceAsText() string
	WriteStackTraceTo(writer io.Writer) error
}

// CaptureStack captures whole stack
// When (skipFrames) are more than stack depth then only "created by" entry will be returned
func CaptureStack(skipFrames int) StackTrace {
	return byteStackTrace(captureStackByDebug(skipFrames, false))
}

// CaptureStackTop is an optimized version to capture a limited info about top-level stack entry only
// When (skipFrames) are more than stack depth then only "created by" entry will be returned
func CaptureStackTop(skipFrames int) StackTrace {
	return byteStackTrace(captureStackByDebug(skipFrames, true))
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
	if limitFrames {
		// provides a bit less info, but is 10x times faster
		result := captureStackByCallers(skipFrames+1, true)
		if len(result) > 0 {
			return result
		}
		// result will be empty at stack top that we can't capture by runtime.Callers() so we will fallback
	}
	return captureStackByDebug(skipFrames, limitFrames)
}

func captureStackByDebug(skipFrames int, limitFrames bool) []byte {
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
	return trimCapacity(capacity, stackBytes)
}

func trimCapacity(actualCapacity int, b []byte) []byte {
	n := len(b) << 1
	if actualCapacity > n || cap(b) > n {
		return append(make([]byte, 0, n), b...)
	}
	return b
}

var frameFileSep = []byte("\n\t")

func indexOfFrame(stackBytes []byte, skipFrames int) int {
	offset := 0
	for ; skipFrames > 0; skipFrames-- {
		prevOffset := offset
		if i := bytes.Index(stackBytes[offset:], frameFileSep); i > 0 {
			offset += i + len(frameFileSep)
			if j := bytes.IndexByte(stackBytes[offset:], '\n'); j > 0 {
				offset += j + 1
				if offset == len(stackBytes) {
					return prevOffset
				}
				continue
			}
			return prevOffset
		}
		return -1
	}
	return offset
}

func captureStackByCallers(skipFrames int, limitFrames bool) []byte {
	var pcs []uintptr
	if limitFrames {
		pcs = make([]uintptr, 1)
	} else {
		pcs = make([]uintptr, 50) // maxStackDepth
	}
	pcs = pcs[:runtime.Callers(skipFrames+2, pcs)]

	result := make([]byte, 0, len(pcs)<<7)
	for i, pc := range pcs {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		fName := fn.Name()
		if i == len(pcs)-1 && fName == "runtime.goexit" {
			continue
		}

		if len(result) > 0 {
			result = append(result, '\n')
		}

		result = append(result, fName...)
		result = append(result, "\n\t"...)

		fName, line := fn.FileLine(pc)
		result = append(result, fName...)
		result = append(result, ':')
		result = strconv.AppendInt(result, int64(line), 10)
	}

	return trimCapacity(0, result)
}
