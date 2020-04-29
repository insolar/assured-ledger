// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"bytes"
)

func MinimizeStackTrace(st StackTrace, boundPackage string, includeBoundary bool) StackTrace {
	var r stackTrace
	switch v := st.(type) {
	case stackTrace:
		r = v
	case nil:
		return nil
	default:
		r.data = []byte(v.StackTraceAsText())
		r.limit = !v.IsFullStack()
	}
	min := MinimizePanicStack(r.data, boundPackage, includeBoundary)
	if len(min) == len(r.data) {
		return st
	}
	r.data = min
	return r
}

func MinimizePanicStack(stackTrace []byte, boundPackage string, includeBoundary bool) []byte {
	s := alignStart(stackTrace)
	if len(s) == 0 {
		return stackTrace
	}
	pos := skipDebugFrame(s)
	s = s[pos:]

	prefix := []byte(boundPackage)
	// check defer section of panic stack - it will likely start with the boundPackage
	start := skipFramesWithPrefix(s, prefix)
	if start > 0 {
		if start == len(s) {
			return s
		}
		n := skipPanicFrame(s[start:])
		if n == 0 {
			if bytes.HasPrefix(s[start:], []byte("created by ")) {
				return s
			}
			return minimizeDebugStack(s[start:], boundPackage, includeBoundary)
		}

		n += start
		n += len(minimizeDebugStack(s[n:], boundPackage, includeBoundary))
		if n == len(s) {
			return s
		}
		return s[start:n]
	}

	start = 0
	pos = 0

	for {
		n := skipPanicFrame(s[pos:])
		if n > 0 {
			pos += n
			n = len(minimizeDebugStack(s[pos:], boundPackage, includeBoundary))
			return s[start : pos+n]
		}

		if bytes.HasPrefix(s[pos:], prefix) {
			break
		}
		n = indexOfFrame(s[pos:], 1)
		if n <= 0 {
			return s
		}
		pos += n
	}

	startBound := pos
	pos += skipFramesWithPrefix(s[pos:], prefix)
	if pos == len(s) {
		return s[start:]
	}

	if !bytes.Contains(s[pos:], []byte("runtime/panic.go")) {
		if startBound == start {
			return s
		}
		if includeBoundary {
			if n := indexOfFrame(s[startBound:], 1); n > 0 {
				startBound += n
			}
		}
		return s[start:startBound]
	}

	return minimizeDebugStack(s[pos:], boundPackage, includeBoundary)
}

func skipPanicFrame(s []byte) int {
	return skipFrameByMethod(s, "panic", "runtime/panic.go")
}

func skipDebugFrame(s []byte) int {
	return skipFrameByMethod(s, "runtime/debug.Stack", "debug/stack.go")
}

func MinimizeDebugStack(stackTrace []byte, boundPackage string, includeBoundary bool) []byte {
	s := alignStart(stackTrace)
	if len(s) == 0 {
		return stackTrace
	}
	pos := skipDebugFrame(s)
	return minimizeDebugStack(s[pos:], boundPackage, includeBoundary)
}

func minimizeDebugStack(s []byte, boundPackage string, includeBoundary bool) []byte {
	end, ok := skipFramesUntilPrefix(s, []byte(boundPackage))
	if end == 0 {
		return s
	}
	if ok && includeBoundary {
		if n := indexOfFrame(s[end:], 1); n > 0 {
			end += n
		}
	}
	return s[:end]
}

func skipFramesUntilPrefix(s []byte, prefix []byte) (int, bool) {
	end := 0
	for {
		if bytes.HasPrefix(s[end:], prefix) {
			return end, true
		}
		switch n := indexOfFrame(s[end:], 1); {
		case n > 0:
			end += n
		case n == 0:
			return len(s), false
		default:
			return end, false
		}
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
	if s[eol+1] != '\t' {
		return 0
	}
	frameEnd := eol
	if n := bytes.IndexByte(s[eol+2:], '\n'); n >= 0 {
		frameEnd += 3 + n
		s = s[:frameEnd]
	} else {
		frameEnd = len(s)
	}
	slash := bytes.LastIndexByte(s, '/')
	if slash < 0 {
		return 0
	}
	if semicolon := bytes.LastIndexByte(s[slash+1:], ':'); semicolon >= 0 {
		s = s[:slash+semicolon+1]
	}
	if !bytes.HasSuffix(s, []byte(fileSuffix)) {
		return 0
	}
	return frameEnd
}
