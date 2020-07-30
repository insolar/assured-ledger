package network

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"

	mm_network "github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// NodeNetworkMock implements network.NodeNetwork
type NodeNetworkMock struct {
	t minimock.Tester

	funcGetAccessor          func(n1 pulse.Number) (a1 mm_network.Accessor)
	inspectFuncGetAccessor   func(n1 pulse.Number)
	afterGetAccessorCounter  uint64
	beforeGetAccessorCounter uint64
	GetAccessorMock          mNodeNetworkMockGetAccessor

	funcGetOrigin          func() (n1 nodeinfo.NetworkNode)
	inspectFuncGetOrigin   func()
	afterGetOriginCounter  uint64
	beforeGetOriginCounter uint64
	GetOriginMock          mNodeNetworkMockGetOrigin
}

// NewNodeNetworkMock returns a mock for network.NodeNetwork
func NewNodeNetworkMock(t minimock.Tester) *NodeNetworkMock {
	m := &NodeNetworkMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.GetAccessorMock = mNodeNetworkMockGetAccessor{mock: m}
	m.GetAccessorMock.callArgs = []*NodeNetworkMockGetAccessorParams{}

	m.GetOriginMock = mNodeNetworkMockGetOrigin{mock: m}

	return m
}

type mNodeNetworkMockGetAccessor struct {
	mock               *NodeNetworkMock
	defaultExpectation *NodeNetworkMockGetAccessorExpectation
	expectations       []*NodeNetworkMockGetAccessorExpectation

	callArgs []*NodeNetworkMockGetAccessorParams
	mutex    sync.RWMutex
}

// NodeNetworkMockGetAccessorExpectation specifies expectation struct of the NodeNetwork.GetAccessor
type NodeNetworkMockGetAccessorExpectation struct {
	mock    *NodeNetworkMock
	params  *NodeNetworkMockGetAccessorParams
	results *NodeNetworkMockGetAccessorResults
	Counter uint64
}

// NodeNetworkMockGetAccessorParams contains parameters of the NodeNetwork.GetAccessor
type NodeNetworkMockGetAccessorParams struct {
	n1 pulse.Number
}

// NodeNetworkMockGetAccessorResults contains results of the NodeNetwork.GetAccessor
type NodeNetworkMockGetAccessorResults struct {
	a1 mm_network.Accessor
}

// Expect sets up expected params for NodeNetwork.GetAccessor
func (mmGetAccessor *mNodeNetworkMockGetAccessor) Expect(n1 pulse.Number) *mNodeNetworkMockGetAccessor {
	if mmGetAccessor.mock.funcGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("NodeNetworkMock.GetAccessor mock is already set by Set")
	}

	if mmGetAccessor.defaultExpectation == nil {
		mmGetAccessor.defaultExpectation = &NodeNetworkMockGetAccessorExpectation{}
	}

	mmGetAccessor.defaultExpectation.params = &NodeNetworkMockGetAccessorParams{n1}
	for _, e := range mmGetAccessor.expectations {
		if minimock.Equal(e.params, mmGetAccessor.defaultExpectation.params) {
			mmGetAccessor.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmGetAccessor.defaultExpectation.params)
		}
	}

	return mmGetAccessor
}

// Inspect accepts an inspector function that has same arguments as the NodeNetwork.GetAccessor
func (mmGetAccessor *mNodeNetworkMockGetAccessor) Inspect(f func(n1 pulse.Number)) *mNodeNetworkMockGetAccessor {
	if mmGetAccessor.mock.inspectFuncGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("Inspect function is already set for NodeNetworkMock.GetAccessor")
	}

	mmGetAccessor.mock.inspectFuncGetAccessor = f

	return mmGetAccessor
}

// Return sets up results that will be returned by NodeNetwork.GetAccessor
func (mmGetAccessor *mNodeNetworkMockGetAccessor) Return(a1 mm_network.Accessor) *NodeNetworkMock {
	if mmGetAccessor.mock.funcGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("NodeNetworkMock.GetAccessor mock is already set by Set")
	}

	if mmGetAccessor.defaultExpectation == nil {
		mmGetAccessor.defaultExpectation = &NodeNetworkMockGetAccessorExpectation{mock: mmGetAccessor.mock}
	}
	mmGetAccessor.defaultExpectation.results = &NodeNetworkMockGetAccessorResults{a1}
	return mmGetAccessor.mock
}

//Set uses given function f to mock the NodeNetwork.GetAccessor method
func (mmGetAccessor *mNodeNetworkMockGetAccessor) Set(f func(n1 pulse.Number) (a1 mm_network.Accessor)) *NodeNetworkMock {
	if mmGetAccessor.defaultExpectation != nil {
		mmGetAccessor.mock.t.Fatalf("Default expectation is already set for the NodeNetwork.GetAccessor method")
	}

	if len(mmGetAccessor.expectations) > 0 {
		mmGetAccessor.mock.t.Fatalf("Some expectations are already set for the NodeNetwork.GetAccessor method")
	}

	mmGetAccessor.mock.funcGetAccessor = f
	return mmGetAccessor.mock
}

// When sets expectation for the NodeNetwork.GetAccessor which will trigger the result defined by the following
// Then helper
func (mmGetAccessor *mNodeNetworkMockGetAccessor) When(n1 pulse.Number) *NodeNetworkMockGetAccessorExpectation {
	if mmGetAccessor.mock.funcGetAccessor != nil {
		mmGetAccessor.mock.t.Fatalf("NodeNetworkMock.GetAccessor mock is already set by Set")
	}

	expectation := &NodeNetworkMockGetAccessorExpectation{
		mock:   mmGetAccessor.mock,
		params: &NodeNetworkMockGetAccessorParams{n1},
	}
	mmGetAccessor.expectations = append(mmGetAccessor.expectations, expectation)
	return expectation
}

// Then sets up NodeNetwork.GetAccessor return parameters for the expectation previously defined by the When method
func (e *NodeNetworkMockGetAccessorExpectation) Then(a1 mm_network.Accessor) *NodeNetworkMock {
	e.results = &NodeNetworkMockGetAccessorResults{a1}
	return e.mock
}

// GetAccessor implements network.NodeNetwork
func (mmGetAccessor *NodeNetworkMock) GetAccessor(n1 pulse.Number) (a1 mm_network.Accessor) {
	mm_atomic.AddUint64(&mmGetAccessor.beforeGetAccessorCounter, 1)
	defer mm_atomic.AddUint64(&mmGetAccessor.afterGetAccessorCounter, 1)

	if mmGetAccessor.inspectFuncGetAccessor != nil {
		mmGetAccessor.inspectFuncGetAccessor(n1)
	}

	mm_params := &NodeNetworkMockGetAccessorParams{n1}

	// Record call args
	mmGetAccessor.GetAccessorMock.mutex.Lock()
	mmGetAccessor.GetAccessorMock.callArgs = append(mmGetAccessor.GetAccessorMock.callArgs, mm_params)
	mmGetAccessor.GetAccessorMock.mutex.Unlock()

	for _, e := range mmGetAccessor.GetAccessorMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.a1
		}
	}

	if mmGetAccessor.GetAccessorMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetAccessor.GetAccessorMock.defaultExpectation.Counter, 1)
		mm_want := mmGetAccessor.GetAccessorMock.defaultExpectation.params
		mm_got := NodeNetworkMockGetAccessorParams{n1}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmGetAccessor.t.Errorf("NodeNetworkMock.GetAccessor got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmGetAccessor.GetAccessorMock.defaultExpectation.results
		if mm_results == nil {
			mmGetAccessor.t.Fatal("No results are set for the NodeNetworkMock.GetAccessor")
		}
		return (*mm_results).a1
	}
	if mmGetAccessor.funcGetAccessor != nil {
		return mmGetAccessor.funcGetAccessor(n1)
	}
	mmGetAccessor.t.Fatalf("Unexpected call to NodeNetworkMock.GetAccessor. %v", n1)
	return
}

// GetAccessorAfterCounter returns a count of finished NodeNetworkMock.GetAccessor invocations
func (mmGetAccessor *NodeNetworkMock) GetAccessorAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetAccessor.afterGetAccessorCounter)
}

// GetAccessorBeforeCounter returns a count of NodeNetworkMock.GetAccessor invocations
func (mmGetAccessor *NodeNetworkMock) GetAccessorBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetAccessor.beforeGetAccessorCounter)
}

// Calls returns a list of arguments used in each call to NodeNetworkMock.GetAccessor.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmGetAccessor *mNodeNetworkMockGetAccessor) Calls() []*NodeNetworkMockGetAccessorParams {
	mmGetAccessor.mutex.RLock()

	argCopy := make([]*NodeNetworkMockGetAccessorParams, len(mmGetAccessor.callArgs))
	copy(argCopy, mmGetAccessor.callArgs)

	mmGetAccessor.mutex.RUnlock()

	return argCopy
}

// MinimockGetAccessorDone returns true if the count of the GetAccessor invocations corresponds
// the number of defined expectations
func (m *NodeNetworkMock) MinimockGetAccessorDone() bool {
	for _, e := range m.GetAccessorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetAccessorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetAccessor != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetAccessorInspect logs each unmet expectation
func (m *NodeNetworkMock) MinimockGetAccessorInspect() {
	for _, e := range m.GetAccessorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to NodeNetworkMock.GetAccessor with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetAccessorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		if m.GetAccessorMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to NodeNetworkMock.GetAccessor")
		} else {
			m.t.Errorf("Expected call to NodeNetworkMock.GetAccessor with params: %#v", *m.GetAccessorMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetAccessor != nil && mm_atomic.LoadUint64(&m.afterGetAccessorCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetAccessor")
	}
}

type mNodeNetworkMockGetOrigin struct {
	mock               *NodeNetworkMock
	defaultExpectation *NodeNetworkMockGetOriginExpectation
	expectations       []*NodeNetworkMockGetOriginExpectation
}

// NodeNetworkMockGetOriginExpectation specifies expectation struct of the NodeNetwork.GetOrigin
type NodeNetworkMockGetOriginExpectation struct {
	mock *NodeNetworkMock

	results *NodeNetworkMockGetOriginResults
	Counter uint64
}

// NodeNetworkMockGetOriginResults contains results of the NodeNetwork.GetOrigin
type NodeNetworkMockGetOriginResults struct {
	n1 nodeinfo.NetworkNode
}

// Expect sets up expected params for NodeNetwork.GetOrigin
func (mmGetOrigin *mNodeNetworkMockGetOrigin) Expect() *mNodeNetworkMockGetOrigin {
	if mmGetOrigin.mock.funcGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("NodeNetworkMock.GetOrigin mock is already set by Set")
	}

	if mmGetOrigin.defaultExpectation == nil {
		mmGetOrigin.defaultExpectation = &NodeNetworkMockGetOriginExpectation{}
	}

	return mmGetOrigin
}

// Inspect accepts an inspector function that has same arguments as the NodeNetwork.GetOrigin
func (mmGetOrigin *mNodeNetworkMockGetOrigin) Inspect(f func()) *mNodeNetworkMockGetOrigin {
	if mmGetOrigin.mock.inspectFuncGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("Inspect function is already set for NodeNetworkMock.GetOrigin")
	}

	mmGetOrigin.mock.inspectFuncGetOrigin = f

	return mmGetOrigin
}

// Return sets up results that will be returned by NodeNetwork.GetOrigin
func (mmGetOrigin *mNodeNetworkMockGetOrigin) Return(n1 nodeinfo.NetworkNode) *NodeNetworkMock {
	if mmGetOrigin.mock.funcGetOrigin != nil {
		mmGetOrigin.mock.t.Fatalf("NodeNetworkMock.GetOrigin mock is already set by Set")
	}

	if mmGetOrigin.defaultExpectation == nil {
		mmGetOrigin.defaultExpectation = &NodeNetworkMockGetOriginExpectation{mock: mmGetOrigin.mock}
	}
	mmGetOrigin.defaultExpectation.results = &NodeNetworkMockGetOriginResults{n1}
	return mmGetOrigin.mock
}

//Set uses given function f to mock the NodeNetwork.GetOrigin method
func (mmGetOrigin *mNodeNetworkMockGetOrigin) Set(f func() (n1 nodeinfo.NetworkNode)) *NodeNetworkMock {
	if mmGetOrigin.defaultExpectation != nil {
		mmGetOrigin.mock.t.Fatalf("Default expectation is already set for the NodeNetwork.GetOrigin method")
	}

	if len(mmGetOrigin.expectations) > 0 {
		mmGetOrigin.mock.t.Fatalf("Some expectations are already set for the NodeNetwork.GetOrigin method")
	}

	mmGetOrigin.mock.funcGetOrigin = f
	return mmGetOrigin.mock
}

// GetOrigin implements network.NodeNetwork
func (mmGetOrigin *NodeNetworkMock) GetOrigin() (n1 nodeinfo.NetworkNode) {
	mm_atomic.AddUint64(&mmGetOrigin.beforeGetOriginCounter, 1)
	defer mm_atomic.AddUint64(&mmGetOrigin.afterGetOriginCounter, 1)

	if mmGetOrigin.inspectFuncGetOrigin != nil {
		mmGetOrigin.inspectFuncGetOrigin()
	}

	if mmGetOrigin.GetOriginMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetOrigin.GetOriginMock.defaultExpectation.Counter, 1)

		mm_results := mmGetOrigin.GetOriginMock.defaultExpectation.results
		if mm_results == nil {
			mmGetOrigin.t.Fatal("No results are set for the NodeNetworkMock.GetOrigin")
		}
		return (*mm_results).n1
	}
	if mmGetOrigin.funcGetOrigin != nil {
		return mmGetOrigin.funcGetOrigin()
	}
	mmGetOrigin.t.Fatalf("Unexpected call to NodeNetworkMock.GetOrigin.")
	return
}

// GetOriginAfterCounter returns a count of finished NodeNetworkMock.GetOrigin invocations
func (mmGetOrigin *NodeNetworkMock) GetOriginAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetOrigin.afterGetOriginCounter)
}

// GetOriginBeforeCounter returns a count of NodeNetworkMock.GetOrigin invocations
func (mmGetOrigin *NodeNetworkMock) GetOriginBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetOrigin.beforeGetOriginCounter)
}

// MinimockGetOriginDone returns true if the count of the GetOrigin invocations corresponds
// the number of defined expectations
func (m *NodeNetworkMock) MinimockGetOriginDone() bool {
	for _, e := range m.GetOriginMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetOriginMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetOrigin != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetOriginInspect logs each unmet expectation
func (m *NodeNetworkMock) MinimockGetOriginInspect() {
	for _, e := range m.GetOriginMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to NodeNetworkMock.GetOrigin")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetOriginMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetOrigin")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetOrigin != nil && mm_atomic.LoadUint64(&m.afterGetOriginCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetOrigin")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *NodeNetworkMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockGetAccessorInspect()

		m.MinimockGetOriginInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *NodeNetworkMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *NodeNetworkMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockGetAccessorDone() &&
		m.MinimockGetOriginDone()
}
