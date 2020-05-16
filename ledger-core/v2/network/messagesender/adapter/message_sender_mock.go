package adapter

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
)

// MessageSenderMock implements MessageSender
type MessageSenderMock struct {
	t minimock.Tester

	funcPrepareAsync          func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc) (a1 smachine.AsyncCallRequester)
	inspectFuncPrepareAsync   func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc)
	afterPrepareAsyncCounter  uint64
	beforePrepareAsyncCounter uint64
	PrepareAsyncMock          mMessageSenderMockPrepareAsync

	funcPrepareNotify          func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) (n1 smachine.NotifyRequester)
	inspectFuncPrepareNotify   func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service))
	afterPrepareNotifyCounter  uint64
	beforePrepareNotifyCounter uint64
	PrepareNotifyMock          mMessageSenderMockPrepareNotify

	funcPrepareSync          func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) (s1 smachine.SyncCallRequester)
	inspectFuncPrepareSync   func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service))
	afterPrepareSyncCounter  uint64
	beforePrepareSyncCounter uint64
	PrepareSyncMock          mMessageSenderMockPrepareSync
}

// NewMessageSenderMock returns a mock for MessageSender
func NewMessageSenderMock(t minimock.Tester) *MessageSenderMock {
	m := &MessageSenderMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.PrepareAsyncMock = mMessageSenderMockPrepareAsync{mock: m}
	m.PrepareAsyncMock.callArgs = []*MessageSenderMockPrepareAsyncParams{}

	m.PrepareNotifyMock = mMessageSenderMockPrepareNotify{mock: m}
	m.PrepareNotifyMock.callArgs = []*MessageSenderMockPrepareNotifyParams{}

	m.PrepareSyncMock = mMessageSenderMockPrepareSync{mock: m}
	m.PrepareSyncMock.callArgs = []*MessageSenderMockPrepareSyncParams{}

	return m
}

type mMessageSenderMockPrepareAsync struct {
	mock               *MessageSenderMock
	defaultExpectation *MessageSenderMockPrepareAsyncExpectation
	expectations       []*MessageSenderMockPrepareAsyncExpectation

	callArgs []*MessageSenderMockPrepareAsyncParams
	mutex    sync.RWMutex
}

// MessageSenderMockPrepareAsyncExpectation specifies expectation struct of the MessageSender.PrepareAsync
type MessageSenderMockPrepareAsyncExpectation struct {
	mock    *MessageSenderMock
	params  *MessageSenderMockPrepareAsyncParams
	results *MessageSenderMockPrepareAsyncResults
	Counter uint64
}

// MessageSenderMockPrepareAsyncParams contains parameters of the MessageSender.PrepareAsync
type MessageSenderMockPrepareAsyncParams struct {
	e1 smachine.ExecutionContext
	fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc
}

// MessageSenderMockPrepareAsyncResults contains results of the MessageSender.PrepareAsync
type MessageSenderMockPrepareAsyncResults struct {
	a1 smachine.AsyncCallRequester
}

// Expect sets up expected params for MessageSender.PrepareAsync
func (mmPrepareAsync *mMessageSenderMockPrepareAsync) Expect(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc) *mMessageSenderMockPrepareAsync {
	if mmPrepareAsync.mock.funcPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("MessageSenderMock.PrepareAsync mock is already set by Set")
	}

	if mmPrepareAsync.defaultExpectation == nil {
		mmPrepareAsync.defaultExpectation = &MessageSenderMockPrepareAsyncExpectation{}
	}

	mmPrepareAsync.defaultExpectation.params = &MessageSenderMockPrepareAsyncParams{e1, fn}
	for _, e := range mmPrepareAsync.expectations {
		if minimock.Equal(e.params, mmPrepareAsync.defaultExpectation.params) {
			mmPrepareAsync.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareAsync.defaultExpectation.params)
		}
	}

	return mmPrepareAsync
}

// Inspect accepts an inspector function that has same arguments as the MessageSender.PrepareAsync
func (mmPrepareAsync *mMessageSenderMockPrepareAsync) Inspect(f func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc)) *mMessageSenderMockPrepareAsync {
	if mmPrepareAsync.mock.inspectFuncPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("Inspect function is already set for MessageSenderMock.PrepareAsync")
	}

	mmPrepareAsync.mock.inspectFuncPrepareAsync = f

	return mmPrepareAsync
}

// Return sets up results that will be returned by MessageSender.PrepareAsync
func (mmPrepareAsync *mMessageSenderMockPrepareAsync) Return(a1 smachine.AsyncCallRequester) *MessageSenderMock {
	if mmPrepareAsync.mock.funcPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("MessageSenderMock.PrepareAsync mock is already set by Set")
	}

	if mmPrepareAsync.defaultExpectation == nil {
		mmPrepareAsync.defaultExpectation = &MessageSenderMockPrepareAsyncExpectation{mock: mmPrepareAsync.mock}
	}
	mmPrepareAsync.defaultExpectation.results = &MessageSenderMockPrepareAsyncResults{a1}
	return mmPrepareAsync.mock
}

//Set uses given function f to mock the MessageSender.PrepareAsync method
func (mmPrepareAsync *mMessageSenderMockPrepareAsync) Set(f func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc) (a1 smachine.AsyncCallRequester)) *MessageSenderMock {
	if mmPrepareAsync.defaultExpectation != nil {
		mmPrepareAsync.mock.t.Fatalf("Default expectation is already set for the MessageSender.PrepareAsync method")
	}

	if len(mmPrepareAsync.expectations) > 0 {
		mmPrepareAsync.mock.t.Fatalf("Some expectations are already set for the MessageSender.PrepareAsync method")
	}

	mmPrepareAsync.mock.funcPrepareAsync = f
	return mmPrepareAsync.mock
}

// When sets expectation for the MessageSender.PrepareAsync which will trigger the result defined by the following
// Then helper
func (mmPrepareAsync *mMessageSenderMockPrepareAsync) When(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc) *MessageSenderMockPrepareAsyncExpectation {
	if mmPrepareAsync.mock.funcPrepareAsync != nil {
		mmPrepareAsync.mock.t.Fatalf("MessageSenderMock.PrepareAsync mock is already set by Set")
	}

	expectation := &MessageSenderMockPrepareAsyncExpectation{
		mock:   mmPrepareAsync.mock,
		params: &MessageSenderMockPrepareAsyncParams{e1, fn},
	}
	mmPrepareAsync.expectations = append(mmPrepareAsync.expectations, expectation)
	return expectation
}

// Then sets up MessageSender.PrepareAsync return parameters for the expectation previously defined by the When method
func (e *MessageSenderMockPrepareAsyncExpectation) Then(a1 smachine.AsyncCallRequester) *MessageSenderMock {
	e.results = &MessageSenderMockPrepareAsyncResults{a1}
	return e.mock
}

// PrepareAsync implements MessageSender
func (mmPrepareAsync *MessageSenderMock) PrepareAsync(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service) smachine.AsyncResultFunc) (a1 smachine.AsyncCallRequester) {
	mm_atomic.AddUint64(&mmPrepareAsync.beforePrepareAsyncCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareAsync.afterPrepareAsyncCounter, 1)

	if mmPrepareAsync.inspectFuncPrepareAsync != nil {
		mmPrepareAsync.inspectFuncPrepareAsync(e1, fn)
	}

	mm_params := &MessageSenderMockPrepareAsyncParams{e1, fn}

	// Record call args
	mmPrepareAsync.PrepareAsyncMock.mutex.Lock()
	mmPrepareAsync.PrepareAsyncMock.callArgs = append(mmPrepareAsync.PrepareAsyncMock.callArgs, mm_params)
	mmPrepareAsync.PrepareAsyncMock.mutex.Unlock()

	for _, e := range mmPrepareAsync.PrepareAsyncMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.a1
		}
	}

	if mmPrepareAsync.PrepareAsyncMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareAsync.PrepareAsyncMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareAsync.PrepareAsyncMock.defaultExpectation.params
		mm_got := MessageSenderMockPrepareAsyncParams{e1, fn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareAsync.t.Errorf("MessageSenderMock.PrepareAsync got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareAsync.PrepareAsyncMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareAsync.t.Fatal("No results are set for the MessageSenderMock.PrepareAsync")
		}
		return (*mm_results).a1
	}
	if mmPrepareAsync.funcPrepareAsync != nil {
		return mmPrepareAsync.funcPrepareAsync(e1, fn)
	}
	mmPrepareAsync.t.Fatalf("Unexpected call to MessageSenderMock.PrepareAsync. %v %v", e1, fn)
	return
}

// PrepareAsyncAfterCounter returns a count of finished MessageSenderMock.PrepareAsync invocations
func (mmPrepareAsync *MessageSenderMock) PrepareAsyncAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareAsync.afterPrepareAsyncCounter)
}

// PrepareAsyncBeforeCounter returns a count of MessageSenderMock.PrepareAsync invocations
func (mmPrepareAsync *MessageSenderMock) PrepareAsyncBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareAsync.beforePrepareAsyncCounter)
}

// Calls returns a list of arguments used in each call to MessageSenderMock.PrepareAsync.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareAsync *mMessageSenderMockPrepareAsync) Calls() []*MessageSenderMockPrepareAsyncParams {
	mmPrepareAsync.mutex.RLock()

	argCopy := make([]*MessageSenderMockPrepareAsyncParams, len(mmPrepareAsync.callArgs))
	copy(argCopy, mmPrepareAsync.callArgs)

	mmPrepareAsync.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareAsyncDone returns true if the count of the PrepareAsync invocations corresponds
// the number of defined expectations
func (m *MessageSenderMock) MinimockPrepareAsyncDone() bool {
	for _, e := range m.PrepareAsyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareAsyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareAsync != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareAsyncInspect logs each unmet expectation
func (m *MessageSenderMock) MinimockPrepareAsyncInspect() {
	for _, e := range m.PrepareAsyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to MessageSenderMock.PrepareAsync with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareAsyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		if m.PrepareAsyncMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to MessageSenderMock.PrepareAsync")
		} else {
			m.t.Errorf("Expected call to MessageSenderMock.PrepareAsync with params: %#v", *m.PrepareAsyncMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareAsync != nil && mm_atomic.LoadUint64(&m.afterPrepareAsyncCounter) < 1 {
		m.t.Error("Expected call to MessageSenderMock.PrepareAsync")
	}
}

type mMessageSenderMockPrepareNotify struct {
	mock               *MessageSenderMock
	defaultExpectation *MessageSenderMockPrepareNotifyExpectation
	expectations       []*MessageSenderMockPrepareNotifyExpectation

	callArgs []*MessageSenderMockPrepareNotifyParams
	mutex    sync.RWMutex
}

// MessageSenderMockPrepareNotifyExpectation specifies expectation struct of the MessageSender.PrepareNotify
type MessageSenderMockPrepareNotifyExpectation struct {
	mock    *MessageSenderMock
	params  *MessageSenderMockPrepareNotifyParams
	results *MessageSenderMockPrepareNotifyResults
	Counter uint64
}

// MessageSenderMockPrepareNotifyParams contains parameters of the MessageSender.PrepareNotify
type MessageSenderMockPrepareNotifyParams struct {
	e1 smachine.ExecutionContext
	fn func(ctx context.Context, svc messagesender.Service)
}

// MessageSenderMockPrepareNotifyResults contains results of the MessageSender.PrepareNotify
type MessageSenderMockPrepareNotifyResults struct {
	n1 smachine.NotifyRequester
}

// Expect sets up expected params for MessageSender.PrepareNotify
func (mmPrepareNotify *mMessageSenderMockPrepareNotify) Expect(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) *mMessageSenderMockPrepareNotify {
	if mmPrepareNotify.mock.funcPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("MessageSenderMock.PrepareNotify mock is already set by Set")
	}

	if mmPrepareNotify.defaultExpectation == nil {
		mmPrepareNotify.defaultExpectation = &MessageSenderMockPrepareNotifyExpectation{}
	}

	mmPrepareNotify.defaultExpectation.params = &MessageSenderMockPrepareNotifyParams{e1, fn}
	for _, e := range mmPrepareNotify.expectations {
		if minimock.Equal(e.params, mmPrepareNotify.defaultExpectation.params) {
			mmPrepareNotify.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareNotify.defaultExpectation.params)
		}
	}

	return mmPrepareNotify
}

// Inspect accepts an inspector function that has same arguments as the MessageSender.PrepareNotify
func (mmPrepareNotify *mMessageSenderMockPrepareNotify) Inspect(f func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service))) *mMessageSenderMockPrepareNotify {
	if mmPrepareNotify.mock.inspectFuncPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("Inspect function is already set for MessageSenderMock.PrepareNotify")
	}

	mmPrepareNotify.mock.inspectFuncPrepareNotify = f

	return mmPrepareNotify
}

// Return sets up results that will be returned by MessageSender.PrepareNotify
func (mmPrepareNotify *mMessageSenderMockPrepareNotify) Return(n1 smachine.NotifyRequester) *MessageSenderMock {
	if mmPrepareNotify.mock.funcPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("MessageSenderMock.PrepareNotify mock is already set by Set")
	}

	if mmPrepareNotify.defaultExpectation == nil {
		mmPrepareNotify.defaultExpectation = &MessageSenderMockPrepareNotifyExpectation{mock: mmPrepareNotify.mock}
	}
	mmPrepareNotify.defaultExpectation.results = &MessageSenderMockPrepareNotifyResults{n1}
	return mmPrepareNotify.mock
}

//Set uses given function f to mock the MessageSender.PrepareNotify method
func (mmPrepareNotify *mMessageSenderMockPrepareNotify) Set(f func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) (n1 smachine.NotifyRequester)) *MessageSenderMock {
	if mmPrepareNotify.defaultExpectation != nil {
		mmPrepareNotify.mock.t.Fatalf("Default expectation is already set for the MessageSender.PrepareNotify method")
	}

	if len(mmPrepareNotify.expectations) > 0 {
		mmPrepareNotify.mock.t.Fatalf("Some expectations are already set for the MessageSender.PrepareNotify method")
	}

	mmPrepareNotify.mock.funcPrepareNotify = f
	return mmPrepareNotify.mock
}

// When sets expectation for the MessageSender.PrepareNotify which will trigger the result defined by the following
// Then helper
func (mmPrepareNotify *mMessageSenderMockPrepareNotify) When(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) *MessageSenderMockPrepareNotifyExpectation {
	if mmPrepareNotify.mock.funcPrepareNotify != nil {
		mmPrepareNotify.mock.t.Fatalf("MessageSenderMock.PrepareNotify mock is already set by Set")
	}

	expectation := &MessageSenderMockPrepareNotifyExpectation{
		mock:   mmPrepareNotify.mock,
		params: &MessageSenderMockPrepareNotifyParams{e1, fn},
	}
	mmPrepareNotify.expectations = append(mmPrepareNotify.expectations, expectation)
	return expectation
}

// Then sets up MessageSender.PrepareNotify return parameters for the expectation previously defined by the When method
func (e *MessageSenderMockPrepareNotifyExpectation) Then(n1 smachine.NotifyRequester) *MessageSenderMock {
	e.results = &MessageSenderMockPrepareNotifyResults{n1}
	return e.mock
}

// PrepareNotify implements MessageSender
func (mmPrepareNotify *MessageSenderMock) PrepareNotify(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) (n1 smachine.NotifyRequester) {
	mm_atomic.AddUint64(&mmPrepareNotify.beforePrepareNotifyCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareNotify.afterPrepareNotifyCounter, 1)

	if mmPrepareNotify.inspectFuncPrepareNotify != nil {
		mmPrepareNotify.inspectFuncPrepareNotify(e1, fn)
	}

	mm_params := &MessageSenderMockPrepareNotifyParams{e1, fn}

	// Record call args
	mmPrepareNotify.PrepareNotifyMock.mutex.Lock()
	mmPrepareNotify.PrepareNotifyMock.callArgs = append(mmPrepareNotify.PrepareNotifyMock.callArgs, mm_params)
	mmPrepareNotify.PrepareNotifyMock.mutex.Unlock()

	for _, e := range mmPrepareNotify.PrepareNotifyMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.n1
		}
	}

	if mmPrepareNotify.PrepareNotifyMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareNotify.PrepareNotifyMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareNotify.PrepareNotifyMock.defaultExpectation.params
		mm_got := MessageSenderMockPrepareNotifyParams{e1, fn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareNotify.t.Errorf("MessageSenderMock.PrepareNotify got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareNotify.PrepareNotifyMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareNotify.t.Fatal("No results are set for the MessageSenderMock.PrepareNotify")
		}
		return (*mm_results).n1
	}
	if mmPrepareNotify.funcPrepareNotify != nil {
		return mmPrepareNotify.funcPrepareNotify(e1, fn)
	}
	mmPrepareNotify.t.Fatalf("Unexpected call to MessageSenderMock.PrepareNotify. %v %v", e1, fn)
	return
}

// PrepareNotifyAfterCounter returns a count of finished MessageSenderMock.PrepareNotify invocations
func (mmPrepareNotify *MessageSenderMock) PrepareNotifyAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareNotify.afterPrepareNotifyCounter)
}

// PrepareNotifyBeforeCounter returns a count of MessageSenderMock.PrepareNotify invocations
func (mmPrepareNotify *MessageSenderMock) PrepareNotifyBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareNotify.beforePrepareNotifyCounter)
}

// Calls returns a list of arguments used in each call to MessageSenderMock.PrepareNotify.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareNotify *mMessageSenderMockPrepareNotify) Calls() []*MessageSenderMockPrepareNotifyParams {
	mmPrepareNotify.mutex.RLock()

	argCopy := make([]*MessageSenderMockPrepareNotifyParams, len(mmPrepareNotify.callArgs))
	copy(argCopy, mmPrepareNotify.callArgs)

	mmPrepareNotify.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareNotifyDone returns true if the count of the PrepareNotify invocations corresponds
// the number of defined expectations
func (m *MessageSenderMock) MinimockPrepareNotifyDone() bool {
	for _, e := range m.PrepareNotifyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareNotifyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareNotify != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareNotifyInspect logs each unmet expectation
func (m *MessageSenderMock) MinimockPrepareNotifyInspect() {
	for _, e := range m.PrepareNotifyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to MessageSenderMock.PrepareNotify with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareNotifyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		if m.PrepareNotifyMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to MessageSenderMock.PrepareNotify")
		} else {
			m.t.Errorf("Expected call to MessageSenderMock.PrepareNotify with params: %#v", *m.PrepareNotifyMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareNotify != nil && mm_atomic.LoadUint64(&m.afterPrepareNotifyCounter) < 1 {
		m.t.Error("Expected call to MessageSenderMock.PrepareNotify")
	}
}

type mMessageSenderMockPrepareSync struct {
	mock               *MessageSenderMock
	defaultExpectation *MessageSenderMockPrepareSyncExpectation
	expectations       []*MessageSenderMockPrepareSyncExpectation

	callArgs []*MessageSenderMockPrepareSyncParams
	mutex    sync.RWMutex
}

// MessageSenderMockPrepareSyncExpectation specifies expectation struct of the MessageSender.PrepareSync
type MessageSenderMockPrepareSyncExpectation struct {
	mock    *MessageSenderMock
	params  *MessageSenderMockPrepareSyncParams
	results *MessageSenderMockPrepareSyncResults
	Counter uint64
}

// MessageSenderMockPrepareSyncParams contains parameters of the MessageSender.PrepareSync
type MessageSenderMockPrepareSyncParams struct {
	e1 smachine.ExecutionContext
	fn func(ctx context.Context, svc messagesender.Service)
}

// MessageSenderMockPrepareSyncResults contains results of the MessageSender.PrepareSync
type MessageSenderMockPrepareSyncResults struct {
	s1 smachine.SyncCallRequester
}

// Expect sets up expected params for MessageSender.PrepareSync
func (mmPrepareSync *mMessageSenderMockPrepareSync) Expect(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) *mMessageSenderMockPrepareSync {
	if mmPrepareSync.mock.funcPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("MessageSenderMock.PrepareSync mock is already set by Set")
	}

	if mmPrepareSync.defaultExpectation == nil {
		mmPrepareSync.defaultExpectation = &MessageSenderMockPrepareSyncExpectation{}
	}

	mmPrepareSync.defaultExpectation.params = &MessageSenderMockPrepareSyncParams{e1, fn}
	for _, e := range mmPrepareSync.expectations {
		if minimock.Equal(e.params, mmPrepareSync.defaultExpectation.params) {
			mmPrepareSync.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPrepareSync.defaultExpectation.params)
		}
	}

	return mmPrepareSync
}

// Inspect accepts an inspector function that has same arguments as the MessageSender.PrepareSync
func (mmPrepareSync *mMessageSenderMockPrepareSync) Inspect(f func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service))) *mMessageSenderMockPrepareSync {
	if mmPrepareSync.mock.inspectFuncPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("Inspect function is already set for MessageSenderMock.PrepareSync")
	}

	mmPrepareSync.mock.inspectFuncPrepareSync = f

	return mmPrepareSync
}

// Return sets up results that will be returned by MessageSender.PrepareSync
func (mmPrepareSync *mMessageSenderMockPrepareSync) Return(s1 smachine.SyncCallRequester) *MessageSenderMock {
	if mmPrepareSync.mock.funcPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("MessageSenderMock.PrepareSync mock is already set by Set")
	}

	if mmPrepareSync.defaultExpectation == nil {
		mmPrepareSync.defaultExpectation = &MessageSenderMockPrepareSyncExpectation{mock: mmPrepareSync.mock}
	}
	mmPrepareSync.defaultExpectation.results = &MessageSenderMockPrepareSyncResults{s1}
	return mmPrepareSync.mock
}

//Set uses given function f to mock the MessageSender.PrepareSync method
func (mmPrepareSync *mMessageSenderMockPrepareSync) Set(f func(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) (s1 smachine.SyncCallRequester)) *MessageSenderMock {
	if mmPrepareSync.defaultExpectation != nil {
		mmPrepareSync.mock.t.Fatalf("Default expectation is already set for the MessageSender.PrepareSync method")
	}

	if len(mmPrepareSync.expectations) > 0 {
		mmPrepareSync.mock.t.Fatalf("Some expectations are already set for the MessageSender.PrepareSync method")
	}

	mmPrepareSync.mock.funcPrepareSync = f
	return mmPrepareSync.mock
}

// When sets expectation for the MessageSender.PrepareSync which will trigger the result defined by the following
// Then helper
func (mmPrepareSync *mMessageSenderMockPrepareSync) When(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) *MessageSenderMockPrepareSyncExpectation {
	if mmPrepareSync.mock.funcPrepareSync != nil {
		mmPrepareSync.mock.t.Fatalf("MessageSenderMock.PrepareSync mock is already set by Set")
	}

	expectation := &MessageSenderMockPrepareSyncExpectation{
		mock:   mmPrepareSync.mock,
		params: &MessageSenderMockPrepareSyncParams{e1, fn},
	}
	mmPrepareSync.expectations = append(mmPrepareSync.expectations, expectation)
	return expectation
}

// Then sets up MessageSender.PrepareSync return parameters for the expectation previously defined by the When method
func (e *MessageSenderMockPrepareSyncExpectation) Then(s1 smachine.SyncCallRequester) *MessageSenderMock {
	e.results = &MessageSenderMockPrepareSyncResults{s1}
	return e.mock
}

// PrepareSync implements MessageSender
func (mmPrepareSync *MessageSenderMock) PrepareSync(e1 smachine.ExecutionContext, fn func(ctx context.Context, svc messagesender.Service)) (s1 smachine.SyncCallRequester) {
	mm_atomic.AddUint64(&mmPrepareSync.beforePrepareSyncCounter, 1)
	defer mm_atomic.AddUint64(&mmPrepareSync.afterPrepareSyncCounter, 1)

	if mmPrepareSync.inspectFuncPrepareSync != nil {
		mmPrepareSync.inspectFuncPrepareSync(e1, fn)
	}

	mm_params := &MessageSenderMockPrepareSyncParams{e1, fn}

	// Record call args
	mmPrepareSync.PrepareSyncMock.mutex.Lock()
	mmPrepareSync.PrepareSyncMock.callArgs = append(mmPrepareSync.PrepareSyncMock.callArgs, mm_params)
	mmPrepareSync.PrepareSyncMock.mutex.Unlock()

	for _, e := range mmPrepareSync.PrepareSyncMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.s1
		}
	}

	if mmPrepareSync.PrepareSyncMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPrepareSync.PrepareSyncMock.defaultExpectation.Counter, 1)
		mm_want := mmPrepareSync.PrepareSyncMock.defaultExpectation.params
		mm_got := MessageSenderMockPrepareSyncParams{e1, fn}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPrepareSync.t.Errorf("MessageSenderMock.PrepareSync got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPrepareSync.PrepareSyncMock.defaultExpectation.results
		if mm_results == nil {
			mmPrepareSync.t.Fatal("No results are set for the MessageSenderMock.PrepareSync")
		}
		return (*mm_results).s1
	}
	if mmPrepareSync.funcPrepareSync != nil {
		return mmPrepareSync.funcPrepareSync(e1, fn)
	}
	mmPrepareSync.t.Fatalf("Unexpected call to MessageSenderMock.PrepareSync. %v %v", e1, fn)
	return
}

// PrepareSyncAfterCounter returns a count of finished MessageSenderMock.PrepareSync invocations
func (mmPrepareSync *MessageSenderMock) PrepareSyncAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareSync.afterPrepareSyncCounter)
}

// PrepareSyncBeforeCounter returns a count of MessageSenderMock.PrepareSync invocations
func (mmPrepareSync *MessageSenderMock) PrepareSyncBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPrepareSync.beforePrepareSyncCounter)
}

// Calls returns a list of arguments used in each call to MessageSenderMock.PrepareSync.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPrepareSync *mMessageSenderMockPrepareSync) Calls() []*MessageSenderMockPrepareSyncParams {
	mmPrepareSync.mutex.RLock()

	argCopy := make([]*MessageSenderMockPrepareSyncParams, len(mmPrepareSync.callArgs))
	copy(argCopy, mmPrepareSync.callArgs)

	mmPrepareSync.mutex.RUnlock()

	return argCopy
}

// MinimockPrepareSyncDone returns true if the count of the PrepareSync invocations corresponds
// the number of defined expectations
func (m *MessageSenderMock) MinimockPrepareSyncDone() bool {
	for _, e := range m.PrepareSyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareSyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareSync != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		return false
	}
	return true
}

// MinimockPrepareSyncInspect logs each unmet expectation
func (m *MessageSenderMock) MinimockPrepareSyncInspect() {
	for _, e := range m.PrepareSyncMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to MessageSenderMock.PrepareSync with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PrepareSyncMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		if m.PrepareSyncMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to MessageSenderMock.PrepareSync")
		} else {
			m.t.Errorf("Expected call to MessageSenderMock.PrepareSync with params: %#v", *m.PrepareSyncMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPrepareSync != nil && mm_atomic.LoadUint64(&m.afterPrepareSyncCounter) < 1 {
		m.t.Error("Expected call to MessageSenderMock.PrepareSync")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *MessageSenderMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockPrepareAsyncInspect()

		m.MinimockPrepareNotifyInspect()

		m.MinimockPrepareSyncInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *MessageSenderMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *MessageSenderMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockPrepareAsyncDone() &&
		m.MinimockPrepareNotifyDone() &&
		m.MinimockPrepareSyncDone()
}
