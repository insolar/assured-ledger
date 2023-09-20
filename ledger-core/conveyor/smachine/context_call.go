package smachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

const (
	adapterSyncCallContext     = 1
	adapterAsyncCallContext    = 2
	adapterCallContextDisposed = 3
)

type adapterCallRequest struct {
	ctx       *executionContext
	fn        AdapterCallFunc
	adapterID AdapterID
	executor  AdapterExecutor
	mode      uint8

	flags     AsyncCallFlags
	nestedFn  CreateFactoryFunc
	cancel    *synckit.ChainedCancel
	isLogging bool
}

func (c *adapterCallRequest) discard() {
	c.mode = adapterCallContextDisposed
}

func (c *adapterCallRequest) ensureMode(mode uint8) {
	if c.mode != mode {
		panic("illegal state")
	}
	c.ctx.ensureValid()
}

func (c *adapterCallRequest) ensureValid() {
	if c.mode > 0 && c.mode < adapterCallContextDisposed {
		c.ctx.ensureValid()
		return
	}
	panic("illegal state")
}

func (c *adapterCallRequest) WithCancel(fn *context.CancelFunc) AsyncCallRequester {
	c.ensureValid()

	if c.cancel != nil {
		*fn = c.cancel.Cancel
		return c
	}

	r := *c
	r.cancel = synckit.NewChainedCancel()
	*fn = r.cancel.Cancel
	return &r
}

func (c *adapterCallRequest) WithLog(isLogging bool) AsyncCallRequester {
	c.ensureValid()
	r := *c
	r.isLogging = isLogging
	return &r
}

func (c *adapterCallRequest) WithNested(nestedFn CreateFactoryFunc) AsyncCallRequester {
	c.ensureValid()

	r := *c
	r.nestedFn = nestedFn
	return &r
}

func (c *adapterCallRequest) WithFlags(flags AsyncCallFlags) AsyncCallRequester {
	c.ensureValid()

	r := *c
	r.flags = flags
	return &r
}

func (c *adapterCallRequest) WithoutAutoWakeUp() AsyncCallRequester {
	c.ensureValid()

	r := *c
	r.flags &^= AutoWakeUp
	return &r

}

func (c *adapterCallRequest) Start() {
	c.ensureMode(adapterAsyncCallContext)
	defer c.discard()

	c._startAsync()
}

func (c *adapterCallRequest) DelayedStart() CallConditionalBuilder {
	c.ensureMode(adapterAsyncCallContext)
	defer c.discard()

	return callConditionalBuilder{c.ctx, c._startAsync}
}

// WARNING! can be called OUTSIDE of context validity (for DelayedStart)
func (c *adapterCallRequest) _startAsync() {
	if c.cancel != nil && c.cancel.IsCancelled() {
		return
	}

	var localCallID uint16 // to explicitly control type
	localCallID = c.ctx.countAsyncCalls

	c.ctx.countAsyncCalls++
	if c.ctx.countAsyncCalls == 0 {
		panic("overflow")
	}

	goCtx := c.ctx._getLoggerCtx()
	var overrideFn AdapterCallbackFunc
	if c.isLogging {
		logger, stepNo := c.ctx._newLoggerAsync()
		callID := uint64(stepNo)<<16 | uint64(localCallID)

		overrideFn = func(isValid bool, resultFunc AsyncResultFunc, err error) bool {
			flags := StepLoggerAdapterAsyncResult
			switch {
			case err != nil || resultFunc != nil:
				if !isValid {
					flags = StepLoggerAdapterAsyncExpiredResult
				}
			case isValid:
				flags = StepLoggerAdapterAsyncCancel
			default:
				flags = StepLoggerAdapterAsyncExpiredCancel
			}
			logger.adapterCall(flags, c.adapterID, callID, err)
			return false // don't stop a valid callback, is ignored for invalid anyway
		}
		logger.adapterCall(StepLoggerAdapterAsyncCall, c.adapterID, callID, nil)
	}

	stepLink := c.ctx.s.NewStepLink()
	callback := NewAdapterCallback(c.adapterID, stepLink, overrideFn, c.flags, c.nestedFn)
	cancelFn := c.executor.StartCall(goCtx, c.fn, callback, c.cancel != nil)

	if c.cancel != nil {
		c.cancel.SetChain(cancelFn)
	}
}

/* ============================================================== */

type adapterSyncCallRequest struct {
	adapterCallRequest
}

func (c *adapterSyncCallRequest) WithLog(isLogging bool) SyncCallRequester {
	c.ensureValid()
	r := *c
	r.isLogging = isLogging
	return &r
}

func (c *adapterSyncCallRequest) WithNested(nestedFn CreateFactoryFunc) SyncCallRequester {
	c.ensureValid()
	r := *c
	r.nestedFn = nestedFn
	return &r
}

func (c *adapterSyncCallRequest) TryCall() bool {
	c.ensureMode(adapterSyncCallContext)
	defer c.discard()

	return c._startSync(true)
}

func (c *adapterSyncCallRequest) Call() {
	c.ensureMode(adapterSyncCallContext)
	defer c.discard()

	if !c._startSync(false) {
		panic("call was cancelled")
	}
}

func (c *adapterSyncCallRequest) _startSync(isTry bool) bool {
	resultFn := c._startSyncWithResult(isTry)

	if resultFn == nil {
		return false
	}

	rc := asyncResultContext{s: c.ctx.s, w: c.ctx.w}
	rc.executeResult(resultFn)
	return true
}

func (c *adapterSyncCallRequest) _startSyncWithResult(isTry bool) AsyncResultFunc {
	c.ctx.ensureValid()

	if c.isLogging {
		logger := c.ctx._newLogger()
		logger.adapterCall(StepLoggerAdapterSyncCall, c.adapterID, 0, nil)
	}

	goCtx := c.ctx._getLoggerCtx()
	if ok, result := c.executor.TrySyncCall(goCtx, c.fn); ok {
		return result
	}

	workerMark := c.ctx.w.GetSignalMark()
	if workerMark == nil {
		return nil
	}

	type resultType struct {
		fn  AsyncResultFunc
		err error
	}
	resultCh := make(chan resultType, 1)

	callback := NewAdapterCallback(c.adapterID, c.ctx.s.NewStepLink(), func(isValid bool, fn AsyncResultFunc, err error) bool {
		if isValid {
			resultCh <- resultType{fn, err}
		} else {
			resultCh <- resultType{nil, err}
		}
		close(resultCh) // prevent repeated callbacks
		return true // stop further processing
	}, 0, c.nestedFn)

	cancelFn := c.executor.StartCall(goCtx, c.fn, callback, false)

	select {
	case result := <-resultCh:
		if result.err != nil {
			panic(result.err)
		}
		return result.fn

	case <-workerMark.ChannelIf(c.ctx.flags&IgnoreSignal == 0, nil):
		if cancelFn != nil {
			cancelFn()
		}
		if isTry {
			return nil
		}
		panic("signal")

	case <-c.ctx.s.machine.GetStoppingSignal():
		if cancelFn != nil {
			cancelFn()
		}
		return nil
	}
}

/* ============================================================== */

type adapterNotifyRequest struct {
	ctx       *executionContext
	fn        AdapterNotifyFunc
	adapterID AdapterID
	executor  AdapterExecutor
	isLogging bool
	mode      uint8
}

func (c *adapterNotifyRequest) discard() {
	c.mode = adapterCallContextDisposed
}

func (c *adapterNotifyRequest) ensure() {
	if c.mode != adapterAsyncCallContext {
		panic("illegal state")
	}
	c.ctx.ensureValid()
}

func (c *adapterNotifyRequest) Send() {
	c.ensure()
	defer c.discard()

	c._startAsync()
}

func (c *adapterNotifyRequest) WithLog(isLogging bool) NotifyRequester {
	c.ensure()
	r := *c
	r.isLogging = isLogging
	return &r
}

func (c *adapterNotifyRequest) DelayedSend() CallConditionalBuilder {
	c.ensure()
	return callConditionalBuilder{c.ctx, c._startAsync}
}

// WARNING! can be called OUTSIDE of context validity (for DelayedSend)
func (c *adapterNotifyRequest) _startAsync() {
	c.SendFunc()()
}

func (c *adapterNotifyRequest) SendFunc() func() {
	goCtx := c.ctx._getLoggerCtx()
	if !c.isLogging {
		return func() {
			c.executor.SendNotify(goCtx, c.fn)
		}
	}

	logger, _ := c.ctx._newLoggerAsync()

	return func() {
		logger.adapterCall(StepLoggerAdapterNotifyCall, c.adapterID, 0, nil)
		c.executor.SendNotify(goCtx, c.fn)
	}
}

/* ============================================================== */

var _ CallConditionalBuilder = callConditionalBuilder{}

type callConditionalBuilder struct {
	ctx     *executionContext
	kickOff StepPrepareFunc
}

func (v callConditionalBuilder) newConditionalUpdate(updType stateUpdKind) ConditionalBuilder {
	cu := v.ctx.newConditionalUpdate(updType)
	cu.kickOff = v.kickOff
	return &cu
}

func (v callConditionalBuilder) Sleep() ConditionalBuilder {
	return v.newConditionalUpdate(stateUpdSleep)
}

func (v callConditionalBuilder) Poll() ConditionalBuilder {
	return v.newConditionalUpdate(stateUpdPoll)
}

func (v callConditionalBuilder) WaitAny() ConditionalBuilder {
	return v.newConditionalUpdate(stateUpdWaitForEvent)
}

func (v callConditionalBuilder) waitNone() ConditionalBuilder {
	return v.newConditionalUpdate(stateUpdNext)
}

func (v callConditionalBuilder) ThenJump(fn StateFunc) StateUpdate {
	return v.waitNone().ThenJump(fn)
}

func (v callConditionalBuilder) ThenJumpExt(step SlotStep) StateUpdate {
	return v.waitNone().ThenJumpExt(step)
}

func (v callConditionalBuilder) ThenRepeat() StateUpdate {
	return v.waitNone().ThenRepeat()
}
