// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

// TODO Work in progress
func CleanWalk(errChain error, fn func(error, string, StackTrace) bool) bool {
	if fn == nil {
		panic(IllegalValue())
	}
	if errChain == nil {
		return false
	}

	last := cleanWalk{fn: fn}
	hadFirst := false
	if Walk(errChain, func(err error, trace StackTrace) bool {
		if !hadFirst {
			hadFirst = true
			return last.firstStep(err, trace)
		}
		return last.nextStep(err, trace)
	}) {
		return true
	}
	if !hadFirst {
		return false
	}
	return last.lastStep()
}

type cleanWalk struct {
	fn func(error, string, StackTrace) bool

	lastMsg   string
	lastTrace StackTrace
	lastErr   error
	noMsg     bool
	cleanMsg  bool
}

func (w *cleanWalk) firstStep(err error, trace StackTrace) bool {
	w.lastErr = err

	switch vv := err.(type) {
	case bypassWrapper:
		w.noMsg = true
		w.cleanMsg = true
		w.lastMsg = vv.LogString()
	case logStringer:
		w.cleanMsg = true
		w.lastMsg = vv.LogString()
	case nil:
		//
	default:
		w.lastMsg = vv.Error()
	}
	if trace != nil {
		w.lastTrace = trace
	}
	return false
}

func (w *cleanWalk) nextStep(err error, trace StackTrace) bool {
	//prevErr := w.lastErr

	switch vv := err.(type) {
	case bypassWrapper:
		switch {
		case w.noMsg:
			w.lastErr = err
			return w.checkTrace(trace)
		case w.cleanMsg:
			nextMsg := vv.LogString()
			if w.lastMsg == nextMsg {
				w.lastErr = err
				w.noMsg = true
				w.cleanMsg = true
				return w.checkTrace(trace)
			}
			//				result := w.fn(w.lastErr, w.lastMsg, w.lastTrace)
		default:
			return w.cleanupMsg(err, trace, vv.LogString())
		}

	case logStringer:
		switch {
		case w.noMsg:
			w.noMsg = false
			w.lastErr = err
			w.cleanMsg = true
			w.lastMsg = vv.LogString()
			return w.checkTrace(trace)

		case w.cleanMsg:
			nextMsg := vv.LogString()
			if w.lastMsg == nextMsg {
				// result := w.fn(w.lastErr, "", w.lastTrace)
				w.lastErr = err
				return w.checkTrace(trace)
			}
			//				result := w.fn(w.lastErr, w.lastMsg, w.lastTrace)
		default:
			return w.cleanupMsg(err, trace, vv.LogString())
		}
	}

	panic(NotImplemented())
	//switch {
	//case err != nil:
	//	b.WriteString(err.Error())
	//	b.WriteByte('\n')
	//	if trace == nil || !includeStack {
	//		return false
	//	}
	//	b.WriteString(stackTracePrintPrefix)
	//case trace != nil && includeStack:
	//	b.WriteString("<nil>\n" + stackTracePrintPrefix)
	//default:
	//	b.WriteString("<nil>\n")
	//	return false
	//}
	//
	//if err := trace.WriteStackTraceTo(b); err != nil {
	//	panic(err)
	//}
	//b.WriteByte('\n')
	//return false
}

func (w *cleanWalk) lastStep() bool {
	if !w.cleanMsg {
		panic(NotImplemented())
	}
	return w.fn(w.lastErr, w.lastMsg, w.lastTrace)
}

func (w *cleanWalk) checkTrace(trace StackTrace) bool {
	switch {
	case trace == nil:
		//
	case w.lastTrace == nil:
		w.lastTrace = trace
	default:
		switch CompareStackTrace(trace, w.lastTrace) {
		case DifferentStack:
			prevTrace := w.lastTrace
			w.lastTrace = trace
			return w.fn(nil, "", prevTrace)
		case EqualStack, SupersetStack, FullStack:
			w.lastTrace = trace
		}
	}
	return false
}

func (w *cleanWalk) cleanupMsg(err error, trace StackTrace, logString string) bool {
	panic(NotImplemented())
}
