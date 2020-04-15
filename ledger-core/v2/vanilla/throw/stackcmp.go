// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import "bytes"

// StackRelation shows relevance of 2 stacks
type StackRelation int8

const (
	SubsetStack StackRelation = iota - 2
	StackTop
	EqualStack
	FullStack     // vs StackTop
	SupersetStack // vs SubsetStack
	DifferentStack
)

// CompareStackTrace returns a relation between the given stacks.
// Only supports StackTrace instances created by this module, otherwise will return DifferentStack
func CompareStackTrace(st0, st1 StackTrace) StackRelation {
	return CompareStackTraceExt(st0, st1, StrictMatch)
}

// StackCompareMode defines precision of stack trace comparison
type StackCompareMode uint8

const (
	StrictMatch StackCompareMode = iota
	SameLine
	SameMethod
)

// CompareStackTraceExt returns a relation between the given stacks with the given mode/precision.
// Only supports StackTrace instances created by this module, otherwise will return DifferentStack
func CompareStackTraceExt(st0, st1 StackTrace, mode StackCompareMode) StackRelation {
	if bst0, ok := st0.(stackTrace); ok {
		if bst1, ok := st1.(stackTrace); ok {
			switch {
			case bst0.limit == bst1.limit:
				return CompareDebugStackTrace(bst0.data, bst1.data, mode)
			case bst0.limit:
				if isStackTraceTop(bst0.data, bst1.data, mode) {
					return StackTop
				}
			default:
				if isStackTraceTop(bst1.data, bst0.data, mode) {
					return FullStack
				}
			}
		}
	}
	return DifferentStack
}

// CompareDebugStackTrace returns a relation between the stacks produced by debug.Stack() with the given mode/precision.
func CompareDebugStackTrace(bst0, bst1 []byte, mode StackCompareMode) StackRelation {
	// TODO This method will break on 100+ frames and then debug.Stack may output "...additional frames elided..."
	// see runtime.traceback1 in runtime/traceback.go
	if mode == StrictMatch {
		switch {
		case len(bst0) == len(bst1):
			if bytes.Equal(bst0, bst1) {
				return EqualStack
			}
		case len(bst0) > len(bst1):
			if bytes.HasSuffix(bst0, bst1) {
				return SupersetStack
			}
		case bytes.HasSuffix(bst1, bst0):
			return SubsetStack
		}
		return DifferentStack
	}

	swapped := false
	if len(bst0) > len(bst1) {
		swapped = true
		bst0, bst1 = bst1, bst0
	}

	compare := 0
	for {
		lastFrameEnd, lastShortestEq := backwardCmpEol(bst0, bst1)

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
			return DifferentStack
		}
		bst0s := bst0[sepPos+1 : lastFrameEnd]

		d := len(bst1) - len(bst0)
		lastEq := d + lastShortestEq
		lastFrameEnd1 := d + lastFrameEnd
		sepPos1 := bytes.LastIndexByte(bst1[:lastEq], sep)
		if sepPos1 <= 0 {
			return DifferentStack
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
		return EqualStack
	case compare > 0 != swapped:
		return SupersetStack
	default:
		return SubsetStack
	}
}

func backwardCmpEol(bShortest, b2 []byte) (lastFrameEnd int, lastShortestEq int) {
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

func isStackTraceTop(bstTop, bstFull []byte, mode StackCompareMode) bool {
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
