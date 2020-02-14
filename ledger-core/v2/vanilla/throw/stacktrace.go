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
)

const StackTracePrefix = "Stack trace: "

type StackTrace interface {
	StackTraceAsText() string
	WriteStackTraceTo(writer io.Writer) error
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

type StackTraceRelation int8

const (
	SubsetTrace StackTraceRelation = iota - 2
	TopTrace
	EqualTrace
	FullTrace
	SupersetTrace
	DifferentTrace
)

func CompareStackTrace(st0, st1 StackTrace) StackTraceRelation {
	return CompareStackTraceExt(st0, st1, StrictMatch)
}

type StackTraceCompareTolerance uint8

const (
	StrictMatch StackTraceCompareTolerance = iota
	CodeOffsetIgnoredOnTop
	LineIgnoredOnTop
)

func CompareStackTraceExt(st0, st1 StackTrace, mode StackTraceCompareTolerance) StackTraceRelation {
	if bst0, ok := st0.(stackTrace); ok {
		if bst1, ok := st1.(stackTrace); ok {
			switch {
			case bst0.limit == bst1.limit:
				if mode == StrictMatch {
					switch n := len(bst0.data) - len(bst1.data); {
					case n == 0:
						if bytes.Equal(bst0.data, bst1.data) {
							return EqualTrace
						}
					case n > 0:
						if bytes.HasSuffix(bst0.data, bst1.data) {
							return SupersetTrace
						}
					case bytes.HasSuffix(bst1.data, bst0.data):
						return SubsetTrace
					}
					return DifferentTrace
				}
				return compareDebugStackTrace(bst0.data, bst1.data, mode)
			case bst0.limit:
				if isStackTraceTop(bst0.data, bst1.data, mode) {
					return TopTrace
				}
			default:
				if isStackTraceTop(bst1.data, bst0.data, mode) {
					return FullTrace
				}
			}
		}
	}
	return DifferentTrace
}

func compareDebugStackTrace(bst0, bst1 []byte, mode StackTraceCompareTolerance) StackTraceRelation {
	// Here is the problem - the topmost entry may differ by line number or code offset within the same method
	// E.g.:
	//	github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw.errBuilder._err2(0x895980, 0xc00003a040, 0x0, 0x0)
	//		/github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw/chain_test.go:57 +0x66
	// vs
	// github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw.errBuilder._err2(0x895980, 0xc00003a040, 0x0, 0x0)
	//		/github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw/chain_test.go:57 +0x44
	//
	swapped := false
	if len(bst0) > len(bst1) {
		swapped = true
		bst0, bst1 = bst1, bst0
	}

	bst0I := indexOfFrame(bst0, 1)
	if bst0I < 0 {
		// this is the only frame of bst0
		switch _, bst1P, b := isStackTraceTopExt(bst0, bst1, mode); {
		case !b:
			return DifferentTrace
		case bst1P == len(bst1):
			return EqualTrace
		case swapped:
			return FullTrace
		default:
			return TopTrace
		}
	}

	bst0P, bst1P, b := isStackTraceTopExt(bst0[:bst0I], bst1, mode)

	if b {
		if bytes.Equal(bst0[bst0P:], bst1[bst1P:]) {
			return EqualTrace
		}
		return DifferentTrace
	}

	switch {
	case !bytes.HasSuffix(bst1, bst0):
		return DifferentTrace
	case swapped:
		return SupersetTrace
	default:
		return SubsetTrace
	}
}

func isStackTraceTop(bstTop, bstFull []byte, mode StackTraceCompareTolerance) bool {
	_, _, b := isStackTraceTopExt(bstTop, bstFull, mode)
	return b
}

func isStackTraceTopExt(bstTop, bstFull []byte, mode StackTraceCompareTolerance) (int, int, bool) {
	n := len(bstTop)
	if len(bstFull) <= n {
		return 0, 0, false
	}

	i, j := cmpLongestTillEol(bstTop, bstFull), 0
	switch {
	case i == n:
		return i, i, true
	case bstTop[i] != '\n':
		return 0, 0, false
	case bstFull[i] == '(':
		j = bytes.IndexByte(bstFull[i:], '\n')
		if j < 0 {
			return 0, 0, false
		}
	case bstFull[i] != '\n':
		return 0, 0, false
	}

	i++
	k := cmpLongestTillEol(bstTop[i:], bstFull[i+j:])
	j += k
	k += i

	switch {
	case k == n:
		return k, j, true
	case bstTop[k] == '\n':
		return k, j, true
	}

	var sep byte
	switch mode {
	case CodeOffsetIgnoredOnTop:
		sep = '+'
	case LineIgnoredOnTop:
		sep = ':'
	default:
		return 0, 0, false
	}

	z := bytes.IndexByte(bstTop[k:], '\n')
	if z < 0 {
		z = len(bstTop) - 1
	} else {
		z += k
	}
	m := bytes.LastIndexByte(bstTop[i:z], sep)
	if m >= 0 && i+m < k {
		z++
		jj := bytes.IndexByte(bstFull[j:], '\n')
		if jj < 0 {
			jj = len(bstFull)
		} else {
			jj += j
			jj++
		}
		return z, jj, true
	}

	return 0, 0, false
}

func cmpLongestTillEol(shortest, s []byte) int {
	i := 0
	for n := len(shortest); i < n; i++ {
		switch shortest[i] {
		case '\n':
			return i
		case s[i]:
			continue
		}
		break
	}
	return i
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
	return StackTracePrefix + string(v.data)
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
