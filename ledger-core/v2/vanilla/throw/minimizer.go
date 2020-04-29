// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"bytes"
)

func MinimizePanicStack(stackTrace []byte, boundPackage string, includeBoundary bool) []byte {
	s := alignStart(stackTrace)
	if len(s) == 0 {
		return stackTrace
	}
	pos := skipFrameByMethod(s, "runtime/debug.Stack", "debug/stack.go")
	s = s[pos:]

	prefix := []byte(boundPackage)
	// check defer section of panic stack - it will likely start with the boundPackage
	start := skipFramesWithPrefix(s, prefix)
	if start > 0 {
		if start == len(s) {
			return s
		}
		return minimizeDebugStack(s[start:], boundPackage, includeBoundary)
	}

	start = 0
	pos = 0
	// start = indexOfFrame(s, 1)
	// if start <= 0 {
	// 	return s
	// }

	for {
		n := skipFrameByMethod(s, "panic", "runtime/panic.go")

	}
}

func minimizePanicStack(s []byte, boundPackage string) []byte {
}

func MinimizeDebugStack(stackTrace []byte, boundPackage string, includeBoundary bool) []byte {
	s := alignStart(stackTrace)
	if len(s) == 0 {
		return stackTrace
	}
	pos := skipFrameByMethod(s, "runtime/debug.Stack", "debug/stack.go")
	return minimizeDebugStack(s[pos:], boundPackage, includeBoundary)
}

func minimizeDebugStack(s []byte, boundPackage string, includeBoundary bool) []byte {
	end, ok := skipFramesUntilPrefix(s, []byte(boundPackage))
	if ok && includeBoundary {
		if n := indexOfFrame(s[end:], 1); n > 0 {
			end += n
		}
	}
	if end == 0 {
		return s
	}
	return s[:end]
}

func skipFramesUntilPrefix(s []byte, prefix []byte) (int, bool) {
	end := 0
	for {
		if bytes.HasPrefix(s[end:], prefix) {
			return end, true
		}
		n := indexOfFrame(s[end:], 1)
		if n <= 0 {
			return end, false
		}
		end += n
	}
}

func skipFramesWithPrefix(s []byte, prefix []byte) int {
	end := 0
	for {
		if !bytes.HasPrefix(s[end:], prefix) {
			return end
		}
		end += len(prefix)
		n := indexOfFrame(s[end:], 1)
		if n <= 0 {
			return len(s)
		}
		end += n
	}
}

func alignStart(s []byte) []byte {
	for {
		eol := bytes.IndexByte(s, '\n')
		if eol <= 0 || eol == len(s)-1 {
			return nil
		}
		if s[eol+1] == '\t' {
			return s
		}
		s = s[eol+1:]
	}
}

func skipFrameByMethod(s []byte, methodPrefix, fileSuffix string) int {
	if !bytes.HasPrefix(s, []byte(methodPrefix)) {
		return 0
	}
	eol := bytes.IndexByte(s, '\n')
	if eol <= 0 || eol == len(s)-1 {
		return 0
	}
	if s[eol+1] == '\t' {
		return 0
	}
	frameEnd := eol
	if n := bytes.IndexByte(s[eol+2:], '\n'); n >= 0 {
		frameEnd += 2 + n
		s = s[:frameEnd]
	} else {
		frameEnd = len(s)
	}
	slash := bytes.LastIndexByte(s, '/')
	if slash < 0 {
		return 0
	}
	if semicolon := bytes.LastIndexByte(s[slash+1:], ':'); semicolon >= 0 {
		s = s[:slash+semicolon]
	}
	if !bytes.HasSuffix(s, []byte(fileSuffix)) {
		return 0
	}
	return frameEnd
}
