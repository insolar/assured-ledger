package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SubscriberState bool

const (
	RemoveSubscriber SubscriberState = false
	RetainSubscriber SubscriberState = true
)

type SubscriberFunc = func(debuglogger.UpdateEvent) SubscriberState

func Filter(predicateFn func(debuglogger.UpdateEvent) bool, outFn SubscriberFunc) SubscriberFunc {
	return newFilter(predicateFn, false, outFn)
}

func Once(predicateFn func(debuglogger.UpdateEvent) bool, outFn SubscriberFunc) SubscriberFunc {
	return newFilter(predicateFn, true, outFn)
}

func EmitByFilter(predicateFn func(debuglogger.UpdateEvent) bool, canClose bool, outSink synckit.ClosableSignalChannel) SubscriberFunc {
	return newFilterSink(predicateFn, false, canClose, outSink)
}

func EmitOnce(predicateFn func(debuglogger.UpdateEvent) bool, canClose bool, outSink synckit.ClosableSignalChannel) SubscriberFunc {
	return newFilterSink(predicateFn, true, canClose, outSink)
}

func newFilter(predicateFn func(debuglogger.UpdateEvent) bool, once bool, outFn SubscriberFunc) SubscriberFunc {
	switch {
	case outFn == nil:
		panic(throw.IllegalValue())
	case predicateFn == nil:
		return outFn
	default:
		return func(event debuglogger.UpdateEvent) SubscriberState {
			if predicateFn(event) || event.IsEmpty() {
				if once {
					outFn(event)
					return RemoveSubscriber
				}
				return outFn(event)
			}
			return RetainSubscriber
		}
	}
}

func sinkOut(canClose bool, outSink synckit.ClosableSignalChannel) {
	select {
	case outSink <- struct {}{}:
	default:
	}
	if canClose {
		close(outSink)
	}
}

func newFilterSink(predicateFn func(debuglogger.UpdateEvent) bool, once, canClose bool, outSink synckit.ClosableSignalChannel) SubscriberFunc {
	switch {
	case outSink == nil:
		panic(throw.IllegalValue())
	case predicateFn == nil:
		return func(event debuglogger.UpdateEvent) SubscriberState {
			switch {
			case outSink == nil:
				return RemoveSubscriber
			case once || event.IsEmpty():
				out := outSink
				outSink = nil
				sinkOut(canClose, out)
				return RemoveSubscriber
			default:
				sinkOut(false, outSink)
				return RetainSubscriber
			}
		}
	default:
		return func(event debuglogger.UpdateEvent) SubscriberState {
			switch {
			case outSink == nil:
				return RemoveSubscriber
			case event.IsEmpty():
				out := outSink
				outSink = nil
				predicateFn(event)
				sinkOut(canClose, out)
				return RemoveSubscriber
			case !predicateFn(event):
				return RetainSubscriber
			case once:
				out := outSink
				outSink = nil
				sinkOut(canClose, out)
				return RemoveSubscriber
			default:
				sinkOut(false, outSink)
				return RetainSubscriber
			}
		}
	}
}
