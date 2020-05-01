// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"bytes"
	"io"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

const StackTracePrefix = "Stack trace:"
const stackTracePrintPrefix = StackTracePrefix + "\n"

type StackTrace interface {
	StackTraceAsText() string
	WriteStackTraceTo(writer io.Writer) error
	IsFullStack() bool
}

// CaptureStack captures whole stack
// When (skipFrames) are more than stack depth then only "created by" entry will be returned
func CaptureStack(skipFrames int) StackTrace {
	return stackTrace{captureStack(skipFrames+1, false), false}
}

// CaptureStackTop is an optimized version to capture a limited info about top-level stack entry only
// When (skipFrames) are more than stack depth then only "created by" entry will be returned
func CaptureStackTop(skipFrames int) StackTrace {
	return stackTrace{captureStack(skipFrames+1, true), true}
}

func ExtractStackTop(st StackTrace, skipFrames int) StackTrace {
	var data []byte
	switch vv := st.(type) {
	case nil:
		return nil
	case stackTrace:
		if vv.limit {
			return st
		}
		data = vv.data
	default:
		if !vv.IsFullStack() {
			return st
		}
		data = []byte(vv.StackTraceAsText())
	}
	start := indexOfFrame(data, skipFrames)
	if start < 0 || start == 0 && skipFrames > 0 {
		return stackTrace{data, true}
	}
	start += skipPanicFrame(data)
	end := indexOfFrame(data[start:], 1)
	if end > 0 {
		data = data[start : start+end]
	} else {
		data = data[start:]
	}
	return stackTrace{append([]byte(nil), data...), true}
}

func IsInSystemPanic(skipFrames int) bool {
	pc := make([]uintptr, 1)
	if runtime.Callers(skipFrames+2, pc) != 1 {
		return false
	}
	n := runtime.FuncForPC(pc[0]).Name()
	return n == "runtime.preprintpanics"
}

type stackTrace struct {
	data  []byte
	limit bool
}

func (v stackTrace) IsFullStack() bool {
	return !v.limit
}

func (v stackTrace) WriteStackTraceTo(w io.Writer) error {
	_, err := w.Write(v.data)
	return err
}

func (v stackTrace) StackTraceAsText() string {
	return string(v.data)
}

func (v stackTrace) LogString() string {
	return string(v.data)
}

func (v stackTrace) String() string {
	return stackTracePrintPrefix + string(v.data)
}

func captureStack(skipFrames int, limitFrames bool) []byte {
	skipFrames++
	if limitFrames {
		// provides a bit less info, but is 10x times faster
		result := captureStackByCallers(skipFrames, true)
		if len(result) > 0 {
			return result
		}
		// result will be empty at stack top that we can't capture by runtime.Callers() so we will fallback
	}
	return captureStackByDebug(skipFrames, limitFrames)
}

const topFrameLimit = 1 // MUST be 1, otherwise comparison of stack vs stack top may not work properly

func captureStackByDebug(skipFrames int, limitFrames bool) []byte {
	stackBytes := debug.Stack()
	capacity := cap(stackBytes)
	if i := bytes.IndexByte(stackBytes, '\n'); i > 0 {
		stackBytes = stackBytes[i+1:]
	} else {
		// strange result, let be safe
		return stackBytes
	}

	const serviceFrames = 2 //  debug.Stack() + captureStackByDebug()
	if i := indexOfFrame(stackBytes, skipFrames+serviceFrames); i > 0 {
		stackBytes = stackBytes[i:]

		if limitFrames {
			if i := indexOfFrame(stackBytes, topFrameLimit); i > 0 {
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

const frameFileSeparator = "\n\t"

var frameFileSep = []byte(frameFileSeparator)

func captureStackByCallers(skipFrames int, limitFrames bool) []byte {
	var pcs []uintptr
	if limitFrames {
		pcs = make([]uintptr, topFrameLimit)
	} else {
		pcs = make([]uintptr, 50) // maxStackDepth
	}
	const serviceFrames = 2 //  runtime.Callers() + captureStackByCallers()
	pcs = pcs[:runtime.Callers(skipFrames+serviceFrames, pcs)]

	result := make([]byte, 0, len(pcs)<<7)
	for i, pc := range pcs {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		fName := fn.Name()
		if i == len(pcs)-1 && fName == "runtime.goexit" {
			break
		}

		// imitation of debug.Stack() format

		if len(result) > 0 {
			result = append(result, '\n')
		}

		result = append(result, fName...)
		result = append(result, frameFileSeparator...)

		fName, line := fn.FileLine(pc)
		result = append(result, fName...)
		result = append(result, ':')
		result = strconv.AppendInt(result, int64(line), 10)
	}

	return trimCapacity(0, result)
}

func TrimStackTrace(s string) string {
	if i := strings.Index(s, "\n"+stackTracePrintPrefix); i >= 0 {
		return s[:i]
	}
	return s
}
