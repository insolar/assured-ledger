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
	SameLine
	SameMethod
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
				return compareDebugStackTrace2(bst0.data, bst1.data, mode)
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
	if bst0I >= 0 {
		if bst0I == 0 || !bytes.HasSuffix(bst1, bst0[bst0I:]) {
			return DifferentTrace
		}
	} else {
		bst0I = len(bst0)
	}

	bst0P := endOfFrame(bst0[:bst0I], mode)
	if bst0P < 0 {
		return DifferentTrace
	}

	bst1I := len(bst1) - (len(bst0) - bst0I)
	bst1P := endOfFrame(bst1[:bst1I], mode)
	if bst1P < 0 {
		return DifferentTrace
	}

	superset := swapped
	switch {
	case bst0P == bst1P:
		switch {
		case !bytes.Equal(bst0[:bst0P], bst1[:bst1P]):
			return DifferentTrace
		case bst0I == bst1I:
			c := bytes.Compare(bst0[bst0P:bst0I], bst1[bst1P:bst1I])
			if c == 0 {
				return EqualTrace
			}
			superset = c < 0 // smaller values of line/code are considered "superset"
		default:
			superset = bst0I < bst1I // smaller values of line/code are considered "superset"
		}
		superset = superset != swapped
	case bst0P > bst1P:
		panic(Impossible())
	case !bytes.HasSuffix(bst1[:bst1P], bst0[:bst0P]):
		return DifferentTrace
	}

	if superset {
		return SupersetTrace
	} else {
		return SubsetTrace
	}
}

func compareDebugStackTrace2(bst0, bst1 []byte, mode StackTraceCompareTolerance) StackTraceRelation {
	swapped := false
	if len(bst0) > len(bst1) {
		swapped = true
		bst0, bst1 = bst1, bst0
	}

	compare := 0
	for {
		lastFrameEnd, lastShortestEq := backwardCmpEol(bst0, bst1, mode)

		if lastShortestEq == 0 {
			switch {
			case len(bst0) == len(bst1):
			case len(bst0) < len(bst1):
				compare = -1
			default:
				compare = 1
			}
			break
		}

		sep := byte(':')
		sepPos := bytes.LastIndexByte(bst0[:lastFrameEnd], sep)

		switch {
		case sepPos <= 0:
		case sepPos >= lastShortestEq:
			sepPos = -1
		case mode == SameLine:
			sepPos++
			sep = ' '
			if sep2 := bytes.IndexByte(bst0[sepPos:lastFrameEnd], sep); sep2 <= 0 {
				sepPos = -1
			} else {
				sepPos += sep2
				if sepPos >= lastShortestEq {
					sepPos = -1
				}
			}
		}

		if sepPos <= 0 {
			return DifferentTrace
		}
		bst0s := bst0[sepPos+1 : lastFrameEnd]

		d := len(bst1) - len(bst0)
		lastEq := d + lastShortestEq
		lastFrameEnd1 := d + lastFrameEnd
		sepPos1 := bytes.LastIndexByte(bst1[:lastEq], sep)
		if sepPos1 <= 0 {
			return DifferentTrace
		}
		bst1s := bst1[sepPos1+1 : lastFrameEnd1]

		switch {
		case len(bst0s) == len(bst1s):
			compare = -bytes.Compare(bst0s, bst1s)
		case len(bst0s) < len(bst1s):
			compare = 1
		default:
			compare = -1
		}

		bst0 = bst0[:sepPos]
		bst1 = bst1[:sepPos1]
	}

	switch {
	case compare == 0:
		return EqualTrace
	case compare > 0 != swapped:
		return SupersetTrace
	default:
		return SubsetTrace
	}
}

func backwardCmpEol(bShortest, b2 []byte, mode StackTraceCompareTolerance) (lastFrameEnd int, lastShortestEq int) {
	lastFrameEnd = len(bShortest) - 1
	if bShortest[lastFrameEnd] != '\n' {
		lastFrameEnd++
	}

	flip := false
	for i, j := lastFrameEnd-1, len(b2); i >= 0; i-- {
		j--
		b := bShortest[i]
		if b != b2[j] {
			return lastFrameEnd, i + 1
		}
		if b == '\n' {
			if flip {
				flip = false
				lastFrameEnd = i
			} else {
				flip = true
			}
		}
	}
	return lastFrameEnd, 0
}

func endOfFrame(b []byte, mode StackTraceCompareTolerance) int {
	i := bytes.LastIndexByte(b, ':')
	if i <= 0 {
		return -1
	}
	switch mode {
	case SameMethod:
		return i
	case SameLine:
		//
	default:
		panic(IllegalValue())
	}

	i++
	j := bytes.IndexByte(b[i:], ' ')
	if j < 0 {
		return len(b)
	}
	return j + i
}

func isStackTraceTop(bstTop, bstFull []byte, mode StackTraceCompareTolerance) bool {
	n := len(bstTop)
	if len(bstFull) <= n {
		return false
	}

	i, j := cmpLongestTillEol(bstTop, bstFull), 0
	switch {
	case i == n:
		return true
	case bstTop[i] != '\n':
		return false
	case bstFull[i] == '(':
		j = bytes.IndexByte(bstFull[i:], '\n')
		if j < 0 {
			return false
		}
	case bstFull[i] != '\n':
		return false
	}

	i++
	k := cmpLongestTillEol(bstTop[i:], bstFull[i+j:])
	j += k
	k += i

	switch {
	case k == n:
		return true
	case bstTop[k] == '\n':
		return true
	}

	var sep byte
	switch mode {
	case SameLine:
		sep = '+'
	case SameMethod:
		sep = ':'
	default:
		return false
	}

	z := bytes.IndexByte(bstTop[k:], '\n')
	if z < 0 {
		z = len(bstTop) - 1
	} else {
		z += k
	}
	m := bytes.LastIndexByte(bstTop[i:z], sep)
	if m >= 0 && i+m < k {
		//z++
		//jj := bytes.IndexByte(bstFull[j:], '\n')
		//if jj < 0 {
		//	jj = len(bstFull)
		//} else {
		//	jj += j
		//	jj++
		//}
		return true
	}

	return false
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
