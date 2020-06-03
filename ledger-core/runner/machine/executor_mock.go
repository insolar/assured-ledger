package machine

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"context"
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
)

// ExecutorMock implements Executor
type ExecutorMock struct {
	t minimock.Tester

	funcCallConstructor          func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error)
	inspectFuncCallConstructor   func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte)
	afterCallConstructorCounter  uint64
	beforeCallConstructorCounter uint64
	CallConstructorMock          mExecutorMockCallConstructor

	funcCallMethod          func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error)
	inspectFuncCallMethod   func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte)
	afterCallMethodCounter  uint64
	beforeCallMethodCounter uint64
	CallMethodMock          mExecutorMockCallMethod

	funcClassifyMethod          func(ctx context.Context, codeRef reference.Global, method string) (m1 contract.MethodIsolation, err error)
	inspectFuncClassifyMethod   func(ctx context.Context, codeRef reference.Global, method string)
	afterClassifyMethodCounter  uint64
	beforeClassifyMethodCounter uint64
	ClassifyMethodMock          mExecutorMockClassifyMethod
}

// NewExecutorMock returns a mock for Executor
func NewExecutorMock(t minimock.Tester) *ExecutorMock {
	m := &ExecutorMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.CallConstructorMock = mExecutorMockCallConstructor{mock: m}
	m.CallConstructorMock.callArgs = []*ExecutorMockCallConstructorParams{}

	m.CallMethodMock = mExecutorMockCallMethod{mock: m}
	m.CallMethodMock.callArgs = []*ExecutorMockCallMethodParams{}

	m.ClassifyMethodMock = mExecutorMockClassifyMethod{mock: m}
	m.ClassifyMethodMock.callArgs = []*ExecutorMockClassifyMethodParams{}

	return m
}

type mExecutorMockCallConstructor struct {
	mock               *ExecutorMock
	defaultExpectation *ExecutorMockCallConstructorExpectation
	expectations       []*ExecutorMockCallConstructorExpectation

	callArgs []*ExecutorMockCallConstructorParams
	mutex    sync.RWMutex
}

// ExecutorMockCallConstructorExpectation specifies expectation struct of the Executor.CallConstructor
type ExecutorMockCallConstructorExpectation struct {
	mock    *ExecutorMock
	params  *ExecutorMockCallConstructorParams
	results *ExecutorMockCallConstructorResults
	Counter uint64
}

// ExecutorMockCallConstructorParams contains parameters of the Executor.CallConstructor
type ExecutorMockCallConstructorParams struct {
	ctx         context.Context
	callContext *call.LogicContext
	code        reference.Global
	name        string
	args        []byte
}

// ExecutorMockCallConstructorResults contains results of the Executor.CallConstructor
type ExecutorMockCallConstructorResults struct {
	objectState []byte
	result      []byte
	err         error
}

// Expect sets up expected params for Executor.CallConstructor
func (mmCallConstructor *mExecutorMockCallConstructor) Expect(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) *mExecutorMockCallConstructor {
	if mmCallConstructor.mock.funcCallConstructor != nil {
		mmCallConstructor.mock.t.Fatalf("ExecutorMock.CallConstructor mock is already set by Set")
	}

	if mmCallConstructor.defaultExpectation == nil {
		mmCallConstructor.defaultExpectation = &ExecutorMockCallConstructorExpectation{}
	}

	mmCallConstructor.defaultExpectation.params = &ExecutorMockCallConstructorParams{ctx, callContext, code, name, args}
	for _, e := range mmCallConstructor.expectations {
		if minimock.Equal(e.params, mmCallConstructor.defaultExpectation.params) {
			mmCallConstructor.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmCallConstructor.defaultExpectation.params)
		}
	}

	return mmCallConstructor
}

// Inspect accepts an inspector function that has same arguments as the Executor.CallConstructor
func (mmCallConstructor *mExecutorMockCallConstructor) Inspect(f func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte)) *mExecutorMockCallConstructor {
	if mmCallConstructor.mock.inspectFuncCallConstructor != nil {
		mmCallConstructor.mock.t.Fatalf("Inspect function is already set for ExecutorMock.CallConstructor")
	}

	mmCallConstructor.mock.inspectFuncCallConstructor = f

	return mmCallConstructor
}

// Return sets up results that will be returned by Executor.CallConstructor
func (mmCallConstructor *mExecutorMockCallConstructor) Return(objectState []byte, result []byte, err error) *ExecutorMock {
	if mmCallConstructor.mock.funcCallConstructor != nil {
		mmCallConstructor.mock.t.Fatalf("ExecutorMock.CallConstructor mock is already set by Set")
	}

	if mmCallConstructor.defaultExpectation == nil {
		mmCallConstructor.defaultExpectation = &ExecutorMockCallConstructorExpectation{mock: mmCallConstructor.mock}
	}
	mmCallConstructor.defaultExpectation.results = &ExecutorMockCallConstructorResults{objectState, result, err}
	return mmCallConstructor.mock
}

//Set uses given function f to mock the Executor.CallConstructor method
func (mmCallConstructor *mExecutorMockCallConstructor) Set(f func(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error)) *ExecutorMock {
	if mmCallConstructor.defaultExpectation != nil {
		mmCallConstructor.mock.t.Fatalf("Default expectation is already set for the Executor.CallConstructor method")
	}

	if len(mmCallConstructor.expectations) > 0 {
		mmCallConstructor.mock.t.Fatalf("Some expectations are already set for the Executor.CallConstructor method")
	}

	mmCallConstructor.mock.funcCallConstructor = f
	return mmCallConstructor.mock
}

// When sets expectation for the Executor.CallConstructor which will trigger the result defined by the following
// Then helper
func (mmCallConstructor *mExecutorMockCallConstructor) When(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) *ExecutorMockCallConstructorExpectation {
	if mmCallConstructor.mock.funcCallConstructor != nil {
		mmCallConstructor.mock.t.Fatalf("ExecutorMock.CallConstructor mock is already set by Set")
	}

	expectation := &ExecutorMockCallConstructorExpectation{
		mock:   mmCallConstructor.mock,
		params: &ExecutorMockCallConstructorParams{ctx, callContext, code, name, args},
	}
	mmCallConstructor.expectations = append(mmCallConstructor.expectations, expectation)
	return expectation
}

// Then sets up Executor.CallConstructor return parameters for the expectation previously defined by the When method
func (e *ExecutorMockCallConstructorExpectation) Then(objectState []byte, result []byte, err error) *ExecutorMock {
	e.results = &ExecutorMockCallConstructorResults{objectState, result, err}
	return e.mock
}

// CallConstructor implements Executor
func (mmCallConstructor *ExecutorMock) CallConstructor(ctx context.Context, callContext *call.LogicContext, code reference.Global, name string, args []byte) (objectState []byte, result []byte, err error) {
	mm_atomic.AddUint64(&mmCallConstructor.beforeCallConstructorCounter, 1)
	defer mm_atomic.AddUint64(&mmCallConstructor.afterCallConstructorCounter, 1)

	if mmCallConstructor.inspectFuncCallConstructor != nil {
		mmCallConstructor.inspectFuncCallConstructor(ctx, callContext, code, name, args)
	}

	mm_params := &ExecutorMockCallConstructorParams{ctx, callContext, code, name, args}

	// Record call args
	mmCallConstructor.CallConstructorMock.mutex.Lock()
	mmCallConstructor.CallConstructorMock.callArgs = append(mmCallConstructor.CallConstructorMock.callArgs, mm_params)
	mmCallConstructor.CallConstructorMock.mutex.Unlock()

	for _, e := range mmCallConstructor.CallConstructorMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.objectState, e.results.result, e.results.err
		}
	}

	if mmCallConstructor.CallConstructorMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmCallConstructor.CallConstructorMock.defaultExpectation.Counter, 1)
		mm_want := mmCallConstructor.CallConstructorMock.defaultExpectation.params
		mm_got := ExecutorMockCallConstructorParams{ctx, callContext, code, name, args}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmCallConstructor.t.Errorf("ExecutorMock.CallConstructor got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmCallConstructor.CallConstructorMock.defaultExpectation.results
		if mm_results == nil {
			mmCallConstructor.t.Fatal("No results are set for the ExecutorMock.CallConstructor")
		}
		return (*mm_results).objectState, (*mm_results).result, (*mm_results).err
	}
	if mmCallConstructor.funcCallConstructor != nil {
		return mmCallConstructor.funcCallConstructor(ctx, callContext, code, name, args)
	}
	mmCallConstructor.t.Fatalf("Unexpected call to ExecutorMock.CallConstructor. %v %v %v %v %v", ctx, callContext, code, name, args)
	return
}

// CallConstructorAfterCounter returns a count of finished ExecutorMock.CallConstructor invocations
func (mmCallConstructor *ExecutorMock) CallConstructorAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmCallConstructor.afterCallConstructorCounter)
}

// CallConstructorBeforeCounter returns a count of ExecutorMock.CallConstructor invocations
func (mmCallConstructor *ExecutorMock) CallConstructorBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmCallConstructor.beforeCallConstructorCounter)
}

// Calls returns a list of arguments used in each call to ExecutorMock.CallConstructor.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmCallConstructor *mExecutorMockCallConstructor) Calls() []*ExecutorMockCallConstructorParams {
	mmCallConstructor.mutex.RLock()

	argCopy := make([]*ExecutorMockCallConstructorParams, len(mmCallConstructor.callArgs))
	copy(argCopy, mmCallConstructor.callArgs)

	mmCallConstructor.mutex.RUnlock()

	return argCopy
}

// MinimockCallConstructorDone returns true if the count of the CallConstructor invocations corresponds
// the number of defined expectations
func (m *ExecutorMock) MinimockCallConstructorDone() bool {
	for _, e := range m.CallConstructorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.CallConstructorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterCallConstructorCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcCallConstructor != nil && mm_atomic.LoadUint64(&m.afterCallConstructorCounter) < 1 {
		return false
	}
	return true
}

// MinimockCallConstructorInspect logs each unmet expectation
func (m *ExecutorMock) MinimockCallConstructorInspect() {
	for _, e := range m.CallConstructorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ExecutorMock.CallConstructor with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.CallConstructorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterCallConstructorCounter) < 1 {
		if m.CallConstructorMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ExecutorMock.CallConstructor")
		} else {
			m.t.Errorf("Expected call to ExecutorMock.CallConstructor with params: %#v", *m.CallConstructorMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcCallConstructor != nil && mm_atomic.LoadUint64(&m.afterCallConstructorCounter) < 1 {
		m.t.Error("Expected call to ExecutorMock.CallConstructor")
	}
}

type mExecutorMockCallMethod struct {
	mock               *ExecutorMock
	defaultExpectation *ExecutorMockCallMethodExpectation
	expectations       []*ExecutorMockCallMethodExpectation

	callArgs []*ExecutorMockCallMethodParams
	mutex    sync.RWMutex
}

// ExecutorMockCallMethodExpectation specifies expectation struct of the Executor.CallMethod
type ExecutorMockCallMethodExpectation struct {
	mock    *ExecutorMock
	params  *ExecutorMockCallMethodParams
	results *ExecutorMockCallMethodResults
	Counter uint64
}

// ExecutorMockCallMethodParams contains parameters of the Executor.CallMethod
type ExecutorMockCallMethodParams struct {
	ctx         context.Context
	callContext *call.LogicContext
	code        reference.Global
	data        []byte
	method      string
	args        []byte
}

// ExecutorMockCallMethodResults contains results of the Executor.CallMethod
type ExecutorMockCallMethodResults struct {
	newObjectState []byte
	methodResults  []byte
	err            error
}

// Expect sets up expected params for Executor.CallMethod
func (mmCallMethod *mExecutorMockCallMethod) Expect(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) *mExecutorMockCallMethod {
	if mmCallMethod.mock.funcCallMethod != nil {
		mmCallMethod.mock.t.Fatalf("ExecutorMock.CallMethod mock is already set by Set")
	}

	if mmCallMethod.defaultExpectation == nil {
		mmCallMethod.defaultExpectation = &ExecutorMockCallMethodExpectation{}
	}

	mmCallMethod.defaultExpectation.params = &ExecutorMockCallMethodParams{ctx, callContext, code, data, method, args}
	for _, e := range mmCallMethod.expectations {
		if minimock.Equal(e.params, mmCallMethod.defaultExpectation.params) {
			mmCallMethod.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmCallMethod.defaultExpectation.params)
		}
	}

	return mmCallMethod
}

// Inspect accepts an inspector function that has same arguments as the Executor.CallMethod
func (mmCallMethod *mExecutorMockCallMethod) Inspect(f func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte)) *mExecutorMockCallMethod {
	if mmCallMethod.mock.inspectFuncCallMethod != nil {
		mmCallMethod.mock.t.Fatalf("Inspect function is already set for ExecutorMock.CallMethod")
	}

	mmCallMethod.mock.inspectFuncCallMethod = f

	return mmCallMethod
}

// Return sets up results that will be returned by Executor.CallMethod
func (mmCallMethod *mExecutorMockCallMethod) Return(newObjectState []byte, methodResults []byte, err error) *ExecutorMock {
	if mmCallMethod.mock.funcCallMethod != nil {
		mmCallMethod.mock.t.Fatalf("ExecutorMock.CallMethod mock is already set by Set")
	}

	if mmCallMethod.defaultExpectation == nil {
		mmCallMethod.defaultExpectation = &ExecutorMockCallMethodExpectation{mock: mmCallMethod.mock}
	}
	mmCallMethod.defaultExpectation.results = &ExecutorMockCallMethodResults{newObjectState, methodResults, err}
	return mmCallMethod.mock
}

//Set uses given function f to mock the Executor.CallMethod method
func (mmCallMethod *mExecutorMockCallMethod) Set(f func(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error)) *ExecutorMock {
	if mmCallMethod.defaultExpectation != nil {
		mmCallMethod.mock.t.Fatalf("Default expectation is already set for the Executor.CallMethod method")
	}

	if len(mmCallMethod.expectations) > 0 {
		mmCallMethod.mock.t.Fatalf("Some expectations are already set for the Executor.CallMethod method")
	}

	mmCallMethod.mock.funcCallMethod = f
	return mmCallMethod.mock
}

// When sets expectation for the Executor.CallMethod which will trigger the result defined by the following
// Then helper
func (mmCallMethod *mExecutorMockCallMethod) When(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) *ExecutorMockCallMethodExpectation {
	if mmCallMethod.mock.funcCallMethod != nil {
		mmCallMethod.mock.t.Fatalf("ExecutorMock.CallMethod mock is already set by Set")
	}

	expectation := &ExecutorMockCallMethodExpectation{
		mock:   mmCallMethod.mock,
		params: &ExecutorMockCallMethodParams{ctx, callContext, code, data, method, args},
	}
	mmCallMethod.expectations = append(mmCallMethod.expectations, expectation)
	return expectation
}

// Then sets up Executor.CallMethod return parameters for the expectation previously defined by the When method
func (e *ExecutorMockCallMethodExpectation) Then(newObjectState []byte, methodResults []byte, err error) *ExecutorMock {
	e.results = &ExecutorMockCallMethodResults{newObjectState, methodResults, err}
	return e.mock
}

// CallMethod implements Executor
func (mmCallMethod *ExecutorMock) CallMethod(ctx context.Context, callContext *call.LogicContext, code reference.Global, data []byte, method string, args []byte) (newObjectState []byte, methodResults []byte, err error) {
	mm_atomic.AddUint64(&mmCallMethod.beforeCallMethodCounter, 1)
	defer mm_atomic.AddUint64(&mmCallMethod.afterCallMethodCounter, 1)

	if mmCallMethod.inspectFuncCallMethod != nil {
		mmCallMethod.inspectFuncCallMethod(ctx, callContext, code, data, method, args)
	}

	mm_params := &ExecutorMockCallMethodParams{ctx, callContext, code, data, method, args}

	// Record call args
	mmCallMethod.CallMethodMock.mutex.Lock()
	mmCallMethod.CallMethodMock.callArgs = append(mmCallMethod.CallMethodMock.callArgs, mm_params)
	mmCallMethod.CallMethodMock.mutex.Unlock()

	for _, e := range mmCallMethod.CallMethodMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.newObjectState, e.results.methodResults, e.results.err
		}
	}

	if mmCallMethod.CallMethodMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmCallMethod.CallMethodMock.defaultExpectation.Counter, 1)
		mm_want := mmCallMethod.CallMethodMock.defaultExpectation.params
		mm_got := ExecutorMockCallMethodParams{ctx, callContext, code, data, method, args}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmCallMethod.t.Errorf("ExecutorMock.CallMethod got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmCallMethod.CallMethodMock.defaultExpectation.results
		if mm_results == nil {
			mmCallMethod.t.Fatal("No results are set for the ExecutorMock.CallMethod")
		}
		return (*mm_results).newObjectState, (*mm_results).methodResults, (*mm_results).err
	}
	if mmCallMethod.funcCallMethod != nil {
		return mmCallMethod.funcCallMethod(ctx, callContext, code, data, method, args)
	}
	mmCallMethod.t.Fatalf("Unexpected call to ExecutorMock.CallMethod. %v %v %v %v %v %v", ctx, callContext, code, data, method, args)
	return
}

// CallMethodAfterCounter returns a count of finished ExecutorMock.CallMethod invocations
func (mmCallMethod *ExecutorMock) CallMethodAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmCallMethod.afterCallMethodCounter)
}

// CallMethodBeforeCounter returns a count of ExecutorMock.CallMethod invocations
func (mmCallMethod *ExecutorMock) CallMethodBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmCallMethod.beforeCallMethodCounter)
}

// Calls returns a list of arguments used in each call to ExecutorMock.CallMethod.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmCallMethod *mExecutorMockCallMethod) Calls() []*ExecutorMockCallMethodParams {
	mmCallMethod.mutex.RLock()

	argCopy := make([]*ExecutorMockCallMethodParams, len(mmCallMethod.callArgs))
	copy(argCopy, mmCallMethod.callArgs)

	mmCallMethod.mutex.RUnlock()

	return argCopy
}

// MinimockCallMethodDone returns true if the count of the CallMethod invocations corresponds
// the number of defined expectations
func (m *ExecutorMock) MinimockCallMethodDone() bool {
	for _, e := range m.CallMethodMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.CallMethodMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterCallMethodCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcCallMethod != nil && mm_atomic.LoadUint64(&m.afterCallMethodCounter) < 1 {
		return false
	}
	return true
}

// MinimockCallMethodInspect logs each unmet expectation
func (m *ExecutorMock) MinimockCallMethodInspect() {
	for _, e := range m.CallMethodMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ExecutorMock.CallMethod with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.CallMethodMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterCallMethodCounter) < 1 {
		if m.CallMethodMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ExecutorMock.CallMethod")
		} else {
			m.t.Errorf("Expected call to ExecutorMock.CallMethod with params: %#v", *m.CallMethodMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcCallMethod != nil && mm_atomic.LoadUint64(&m.afterCallMethodCounter) < 1 {
		m.t.Error("Expected call to ExecutorMock.CallMethod")
	}
}

type mExecutorMockClassifyMethod struct {
	mock               *ExecutorMock
	defaultExpectation *ExecutorMockClassifyMethodExpectation
	expectations       []*ExecutorMockClassifyMethodExpectation

	callArgs []*ExecutorMockClassifyMethodParams
	mutex    sync.RWMutex
}

// ExecutorMockClassifyMethodExpectation specifies expectation struct of the Executor.ClassifyMethod
type ExecutorMockClassifyMethodExpectation struct {
	mock    *ExecutorMock
	params  *ExecutorMockClassifyMethodParams
	results *ExecutorMockClassifyMethodResults
	Counter uint64
}

// ExecutorMockClassifyMethodParams contains parameters of the Executor.ClassifyMethod
type ExecutorMockClassifyMethodParams struct {
	ctx     context.Context
	codeRef reference.Global
	method  string
}

// ExecutorMockClassifyMethodResults contains results of the Executor.ClassifyMethod
type ExecutorMockClassifyMethodResults struct {
	m1  contract.MethodIsolation
	err error
}

// Expect sets up expected params for Executor.ClassifyMethod
func (mmClassifyMethod *mExecutorMockClassifyMethod) Expect(ctx context.Context, codeRef reference.Global, method string) *mExecutorMockClassifyMethod {
	if mmClassifyMethod.mock.funcClassifyMethod != nil {
		mmClassifyMethod.mock.t.Fatalf("ExecutorMock.ClassifyMethod mock is already set by Set")
	}

	if mmClassifyMethod.defaultExpectation == nil {
		mmClassifyMethod.defaultExpectation = &ExecutorMockClassifyMethodExpectation{}
	}

	mmClassifyMethod.defaultExpectation.params = &ExecutorMockClassifyMethodParams{ctx, codeRef, method}
	for _, e := range mmClassifyMethod.expectations {
		if minimock.Equal(e.params, mmClassifyMethod.defaultExpectation.params) {
			mmClassifyMethod.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmClassifyMethod.defaultExpectation.params)
		}
	}

	return mmClassifyMethod
}

// Inspect accepts an inspector function that has same arguments as the Executor.ClassifyMethod
func (mmClassifyMethod *mExecutorMockClassifyMethod) Inspect(f func(ctx context.Context, codeRef reference.Global, method string)) *mExecutorMockClassifyMethod {
	if mmClassifyMethod.mock.inspectFuncClassifyMethod != nil {
		mmClassifyMethod.mock.t.Fatalf("Inspect function is already set for ExecutorMock.ClassifyMethod")
	}

	mmClassifyMethod.mock.inspectFuncClassifyMethod = f

	return mmClassifyMethod
}

// Return sets up results that will be returned by Executor.ClassifyMethod
func (mmClassifyMethod *mExecutorMockClassifyMethod) Return(m1 contract.MethodIsolation, err error) *ExecutorMock {
	if mmClassifyMethod.mock.funcClassifyMethod != nil {
		mmClassifyMethod.mock.t.Fatalf("ExecutorMock.ClassifyMethod mock is already set by Set")
	}

	if mmClassifyMethod.defaultExpectation == nil {
		mmClassifyMethod.defaultExpectation = &ExecutorMockClassifyMethodExpectation{mock: mmClassifyMethod.mock}
	}
	mmClassifyMethod.defaultExpectation.results = &ExecutorMockClassifyMethodResults{m1, err}
	return mmClassifyMethod.mock
}

//Set uses given function f to mock the Executor.ClassifyMethod method
func (mmClassifyMethod *mExecutorMockClassifyMethod) Set(f func(ctx context.Context, codeRef reference.Global, method string) (m1 contract.MethodIsolation, err error)) *ExecutorMock {
	if mmClassifyMethod.defaultExpectation != nil {
		mmClassifyMethod.mock.t.Fatalf("Default expectation is already set for the Executor.ClassifyMethod method")
	}

	if len(mmClassifyMethod.expectations) > 0 {
		mmClassifyMethod.mock.t.Fatalf("Some expectations are already set for the Executor.ClassifyMethod method")
	}

	mmClassifyMethod.mock.funcClassifyMethod = f
	return mmClassifyMethod.mock
}

// When sets expectation for the Executor.ClassifyMethod which will trigger the result defined by the following
// Then helper
func (mmClassifyMethod *mExecutorMockClassifyMethod) When(ctx context.Context, codeRef reference.Global, method string) *ExecutorMockClassifyMethodExpectation {
	if mmClassifyMethod.mock.funcClassifyMethod != nil {
		mmClassifyMethod.mock.t.Fatalf("ExecutorMock.ClassifyMethod mock is already set by Set")
	}

	expectation := &ExecutorMockClassifyMethodExpectation{
		mock:   mmClassifyMethod.mock,
		params: &ExecutorMockClassifyMethodParams{ctx, codeRef, method},
	}
	mmClassifyMethod.expectations = append(mmClassifyMethod.expectations, expectation)
	return expectation
}

// Then sets up Executor.ClassifyMethod return parameters for the expectation previously defined by the When method
func (e *ExecutorMockClassifyMethodExpectation) Then(m1 contract.MethodIsolation, err error) *ExecutorMock {
	e.results = &ExecutorMockClassifyMethodResults{m1, err}
	return e.mock
}

// ClassifyMethod implements Executor
func (mmClassifyMethod *ExecutorMock) ClassifyMethod(ctx context.Context, codeRef reference.Global, method string) (m1 contract.MethodIsolation, err error) {
	mm_atomic.AddUint64(&mmClassifyMethod.beforeClassifyMethodCounter, 1)
	defer mm_atomic.AddUint64(&mmClassifyMethod.afterClassifyMethodCounter, 1)

	if mmClassifyMethod.inspectFuncClassifyMethod != nil {
		mmClassifyMethod.inspectFuncClassifyMethod(ctx, codeRef, method)
	}

	mm_params := &ExecutorMockClassifyMethodParams{ctx, codeRef, method}

	// Record call args
	mmClassifyMethod.ClassifyMethodMock.mutex.Lock()
	mmClassifyMethod.ClassifyMethodMock.callArgs = append(mmClassifyMethod.ClassifyMethodMock.callArgs, mm_params)
	mmClassifyMethod.ClassifyMethodMock.mutex.Unlock()

	for _, e := range mmClassifyMethod.ClassifyMethodMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.m1, e.results.err
		}
	}

	if mmClassifyMethod.ClassifyMethodMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmClassifyMethod.ClassifyMethodMock.defaultExpectation.Counter, 1)
		mm_want := mmClassifyMethod.ClassifyMethodMock.defaultExpectation.params
		mm_got := ExecutorMockClassifyMethodParams{ctx, codeRef, method}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmClassifyMethod.t.Errorf("ExecutorMock.ClassifyMethod got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmClassifyMethod.ClassifyMethodMock.defaultExpectation.results
		if mm_results == nil {
			mmClassifyMethod.t.Fatal("No results are set for the ExecutorMock.ClassifyMethod")
		}
		return (*mm_results).m1, (*mm_results).err
	}
	if mmClassifyMethod.funcClassifyMethod != nil {
		return mmClassifyMethod.funcClassifyMethod(ctx, codeRef, method)
	}
	mmClassifyMethod.t.Fatalf("Unexpected call to ExecutorMock.ClassifyMethod. %v %v %v", ctx, codeRef, method)
	return
}

// ClassifyMethodAfterCounter returns a count of finished ExecutorMock.ClassifyMethod invocations
func (mmClassifyMethod *ExecutorMock) ClassifyMethodAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmClassifyMethod.afterClassifyMethodCounter)
}

// ClassifyMethodBeforeCounter returns a count of ExecutorMock.ClassifyMethod invocations
func (mmClassifyMethod *ExecutorMock) ClassifyMethodBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmClassifyMethod.beforeClassifyMethodCounter)
}

// Calls returns a list of arguments used in each call to ExecutorMock.ClassifyMethod.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmClassifyMethod *mExecutorMockClassifyMethod) Calls() []*ExecutorMockClassifyMethodParams {
	mmClassifyMethod.mutex.RLock()

	argCopy := make([]*ExecutorMockClassifyMethodParams, len(mmClassifyMethod.callArgs))
	copy(argCopy, mmClassifyMethod.callArgs)

	mmClassifyMethod.mutex.RUnlock()

	return argCopy
}

// MinimockClassifyMethodDone returns true if the count of the ClassifyMethod invocations corresponds
// the number of defined expectations
func (m *ExecutorMock) MinimockClassifyMethodDone() bool {
	for _, e := range m.ClassifyMethodMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ClassifyMethodMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterClassifyMethodCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcClassifyMethod != nil && mm_atomic.LoadUint64(&m.afterClassifyMethodCounter) < 1 {
		return false
	}
	return true
}

// MinimockClassifyMethodInspect logs each unmet expectation
func (m *ExecutorMock) MinimockClassifyMethodInspect() {
	for _, e := range m.ClassifyMethodMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to ExecutorMock.ClassifyMethod with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ClassifyMethodMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterClassifyMethodCounter) < 1 {
		if m.ClassifyMethodMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to ExecutorMock.ClassifyMethod")
		} else {
			m.t.Errorf("Expected call to ExecutorMock.ClassifyMethod with params: %#v", *m.ClassifyMethodMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcClassifyMethod != nil && mm_atomic.LoadUint64(&m.afterClassifyMethodCounter) < 1 {
		m.t.Error("Expected call to ExecutorMock.ClassifyMethod")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *ExecutorMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockCallConstructorInspect()

		m.MinimockCallMethodInspect()

		m.MinimockClassifyMethodInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *ExecutorMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *ExecutorMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockCallConstructorDone() &&
		m.MinimockCallMethodDone() &&
		m.MinimockClassifyMethodDone()
}
