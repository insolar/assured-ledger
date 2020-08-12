package network

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	mm_network "github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// NodeNetworkMock implements network.NodeNetwork
type NodeNetworkMock struct {
	t minimock.Tester

	funcGetAccessor          func(n1 pulse.Number) (a1 mm_network.Accessor)
	inspectFuncGetAccessor   func(n1 pulse.Number)
	afterGetAccessorCounter  uint64
	beforeGetAccessorCounter uint64
	GetAccessorMock          mNodeNetworkMockGetAccessor

	funcGetLatestAccessor          func() (a1 mm_network.Accessor)
	inspectFuncGetLatestAccessor   func()
	afterGetLatestAccessorCounter  uint64
	beforeGetLatestAccessorCounter uint64
	GetLatestAccessorMock          mNodeNetworkMockGetLatestAccessor

	funcGetLocalNodeReference          func() (h1 reference.Holder)
	inspectFuncGetLocalNodeReference   func()
	afterGetLocalNodeReferenceCounter  uint64
	beforeGetLocalNodeReferenceCounter uint64
	GetLocalNodeReferenceMock          mNodeNetworkMockGetLocalNodeReference

	funcGetLocalNodeRole          func() (p1 member.PrimaryRole)
	inspectFuncGetLocalNodeRole   func()
	afterGetLocalNodeRoleCounter  uint64
	beforeGetLocalNodeRoleCounter uint64
	GetLocalNodeRoleMock          mNodeNetworkMockGetLocalNodeRole

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

	m.GetLatestAccessorMock = mNodeNetworkMockGetLatestAccessor{mock: m}

	m.GetLocalNodeReferenceMock = mNodeNetworkMockGetLocalNodeReference{mock: m}

	m.GetLocalNodeRoleMock = mNodeNetworkMockGetLocalNodeRole{mock: m}

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

type mNodeNetworkMockGetLatestAccessor struct {
	mock               *NodeNetworkMock
	defaultExpectation *NodeNetworkMockGetLatestAccessorExpectation
	expectations       []*NodeNetworkMockGetLatestAccessorExpectation
}

// NodeNetworkMockGetLatestAccessorExpectation specifies expectation struct of the NodeNetwork.GetLatestAccessor
type NodeNetworkMockGetLatestAccessorExpectation struct {
	mock *NodeNetworkMock

	results *NodeNetworkMockGetLatestAccessorResults
	Counter uint64
}

// NodeNetworkMockGetLatestAccessorResults contains results of the NodeNetwork.GetLatestAccessor
type NodeNetworkMockGetLatestAccessorResults struct {
	a1 mm_network.Accessor
}

// Expect sets up expected params for NodeNetwork.GetLatestAccessor
func (mmGetLatestAccessor *mNodeNetworkMockGetLatestAccessor) Expect() *mNodeNetworkMockGetLatestAccessor {
	if mmGetLatestAccessor.mock.funcGetLatestAccessor != nil {
		mmGetLatestAccessor.mock.t.Fatalf("NodeNetworkMock.GetLatestAccessor mock is already set by Set")
	}

	if mmGetLatestAccessor.defaultExpectation == nil {
		mmGetLatestAccessor.defaultExpectation = &NodeNetworkMockGetLatestAccessorExpectation{}
	}

	return mmGetLatestAccessor
}

// Inspect accepts an inspector function that has same arguments as the NodeNetwork.GetLatestAccessor
func (mmGetLatestAccessor *mNodeNetworkMockGetLatestAccessor) Inspect(f func()) *mNodeNetworkMockGetLatestAccessor {
	if mmGetLatestAccessor.mock.inspectFuncGetLatestAccessor != nil {
		mmGetLatestAccessor.mock.t.Fatalf("Inspect function is already set for NodeNetworkMock.GetLatestAccessor")
	}

	mmGetLatestAccessor.mock.inspectFuncGetLatestAccessor = f

	return mmGetLatestAccessor
}

// Return sets up results that will be returned by NodeNetwork.GetLatestAccessor
func (mmGetLatestAccessor *mNodeNetworkMockGetLatestAccessor) Return(a1 mm_network.Accessor) *NodeNetworkMock {
	if mmGetLatestAccessor.mock.funcGetLatestAccessor != nil {
		mmGetLatestAccessor.mock.t.Fatalf("NodeNetworkMock.GetLatestAccessor mock is already set by Set")
	}

	if mmGetLatestAccessor.defaultExpectation == nil {
		mmGetLatestAccessor.defaultExpectation = &NodeNetworkMockGetLatestAccessorExpectation{mock: mmGetLatestAccessor.mock}
	}
	mmGetLatestAccessor.defaultExpectation.results = &NodeNetworkMockGetLatestAccessorResults{a1}
	return mmGetLatestAccessor.mock
}

//Set uses given function f to mock the NodeNetwork.GetLatestAccessor method
func (mmGetLatestAccessor *mNodeNetworkMockGetLatestAccessor) Set(f func() (a1 mm_network.Accessor)) *NodeNetworkMock {
	if mmGetLatestAccessor.defaultExpectation != nil {
		mmGetLatestAccessor.mock.t.Fatalf("Default expectation is already set for the NodeNetwork.GetLatestAccessor method")
	}

	if len(mmGetLatestAccessor.expectations) > 0 {
		mmGetLatestAccessor.mock.t.Fatalf("Some expectations are already set for the NodeNetwork.GetLatestAccessor method")
	}

	mmGetLatestAccessor.mock.funcGetLatestAccessor = f
	return mmGetLatestAccessor.mock
}

// GetLatestAccessor implements network.NodeNetwork
func (mmGetLatestAccessor *NodeNetworkMock) GetLatestAccessor() (a1 mm_network.Accessor) {
	mm_atomic.AddUint64(&mmGetLatestAccessor.beforeGetLatestAccessorCounter, 1)
	defer mm_atomic.AddUint64(&mmGetLatestAccessor.afterGetLatestAccessorCounter, 1)

	if mmGetLatestAccessor.inspectFuncGetLatestAccessor != nil {
		mmGetLatestAccessor.inspectFuncGetLatestAccessor()
	}

	if mmGetLatestAccessor.GetLatestAccessorMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetLatestAccessor.GetLatestAccessorMock.defaultExpectation.Counter, 1)

		mm_results := mmGetLatestAccessor.GetLatestAccessorMock.defaultExpectation.results
		if mm_results == nil {
			mmGetLatestAccessor.t.Fatal("No results are set for the NodeNetworkMock.GetLatestAccessor")
		}
		return (*mm_results).a1
	}
	if mmGetLatestAccessor.funcGetLatestAccessor != nil {
		return mmGetLatestAccessor.funcGetLatestAccessor()
	}
	mmGetLatestAccessor.t.Fatalf("Unexpected call to NodeNetworkMock.GetLatestAccessor.")
	return
}

// GetLatestAccessorAfterCounter returns a count of finished NodeNetworkMock.GetLatestAccessor invocations
func (mmGetLatestAccessor *NodeNetworkMock) GetLatestAccessorAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLatestAccessor.afterGetLatestAccessorCounter)
}

// GetLatestAccessorBeforeCounter returns a count of NodeNetworkMock.GetLatestAccessor invocations
func (mmGetLatestAccessor *NodeNetworkMock) GetLatestAccessorBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLatestAccessor.beforeGetLatestAccessorCounter)
}

// MinimockGetLatestAccessorDone returns true if the count of the GetLatestAccessor invocations corresponds
// the number of defined expectations
func (m *NodeNetworkMock) MinimockGetLatestAccessorDone() bool {
	for _, e := range m.GetLatestAccessorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLatestAccessorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLatestAccessorCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLatestAccessor != nil && mm_atomic.LoadUint64(&m.afterGetLatestAccessorCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetLatestAccessorInspect logs each unmet expectation
func (m *NodeNetworkMock) MinimockGetLatestAccessorInspect() {
	for _, e := range m.GetLatestAccessorMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to NodeNetworkMock.GetLatestAccessor")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLatestAccessorMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLatestAccessorCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetLatestAccessor")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLatestAccessor != nil && mm_atomic.LoadUint64(&m.afterGetLatestAccessorCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetLatestAccessor")
	}
}

type mNodeNetworkMockGetLocalNodeReference struct {
	mock               *NodeNetworkMock
	defaultExpectation *NodeNetworkMockGetLocalNodeReferenceExpectation
	expectations       []*NodeNetworkMockGetLocalNodeReferenceExpectation
}

// NodeNetworkMockGetLocalNodeReferenceExpectation specifies expectation struct of the NodeNetwork.GetLocalNodeReference
type NodeNetworkMockGetLocalNodeReferenceExpectation struct {
	mock *NodeNetworkMock

	results *NodeNetworkMockGetLocalNodeReferenceResults
	Counter uint64
}

// NodeNetworkMockGetLocalNodeReferenceResults contains results of the NodeNetwork.GetLocalNodeReference
type NodeNetworkMockGetLocalNodeReferenceResults struct {
	h1 reference.Holder
}

// Expect sets up expected params for NodeNetwork.GetLocalNodeReference
func (mmGetLocalNodeReference *mNodeNetworkMockGetLocalNodeReference) Expect() *mNodeNetworkMockGetLocalNodeReference {
	if mmGetLocalNodeReference.mock.funcGetLocalNodeReference != nil {
		mmGetLocalNodeReference.mock.t.Fatalf("NodeNetworkMock.GetLocalNodeReference mock is already set by Set")
	}

	if mmGetLocalNodeReference.defaultExpectation == nil {
		mmGetLocalNodeReference.defaultExpectation = &NodeNetworkMockGetLocalNodeReferenceExpectation{}
	}

	return mmGetLocalNodeReference
}

// Inspect accepts an inspector function that has same arguments as the NodeNetwork.GetLocalNodeReference
func (mmGetLocalNodeReference *mNodeNetworkMockGetLocalNodeReference) Inspect(f func()) *mNodeNetworkMockGetLocalNodeReference {
	if mmGetLocalNodeReference.mock.inspectFuncGetLocalNodeReference != nil {
		mmGetLocalNodeReference.mock.t.Fatalf("Inspect function is already set for NodeNetworkMock.GetLocalNodeReference")
	}

	mmGetLocalNodeReference.mock.inspectFuncGetLocalNodeReference = f

	return mmGetLocalNodeReference
}

// Return sets up results that will be returned by NodeNetwork.GetLocalNodeReference
func (mmGetLocalNodeReference *mNodeNetworkMockGetLocalNodeReference) Return(h1 reference.Holder) *NodeNetworkMock {
	if mmGetLocalNodeReference.mock.funcGetLocalNodeReference != nil {
		mmGetLocalNodeReference.mock.t.Fatalf("NodeNetworkMock.GetLocalNodeReference mock is already set by Set")
	}

	if mmGetLocalNodeReference.defaultExpectation == nil {
		mmGetLocalNodeReference.defaultExpectation = &NodeNetworkMockGetLocalNodeReferenceExpectation{mock: mmGetLocalNodeReference.mock}
	}
	mmGetLocalNodeReference.defaultExpectation.results = &NodeNetworkMockGetLocalNodeReferenceResults{h1}
	return mmGetLocalNodeReference.mock
}

//Set uses given function f to mock the NodeNetwork.GetLocalNodeReference method
func (mmGetLocalNodeReference *mNodeNetworkMockGetLocalNodeReference) Set(f func() (h1 reference.Holder)) *NodeNetworkMock {
	if mmGetLocalNodeReference.defaultExpectation != nil {
		mmGetLocalNodeReference.mock.t.Fatalf("Default expectation is already set for the NodeNetwork.GetLocalNodeReference method")
	}

	if len(mmGetLocalNodeReference.expectations) > 0 {
		mmGetLocalNodeReference.mock.t.Fatalf("Some expectations are already set for the NodeNetwork.GetLocalNodeReference method")
	}

	mmGetLocalNodeReference.mock.funcGetLocalNodeReference = f
	return mmGetLocalNodeReference.mock
}

// GetLocalNodeReference implements network.NodeNetwork
func (mmGetLocalNodeReference *NodeNetworkMock) GetLocalNodeReference() (h1 reference.Holder) {
	mm_atomic.AddUint64(&mmGetLocalNodeReference.beforeGetLocalNodeReferenceCounter, 1)
	defer mm_atomic.AddUint64(&mmGetLocalNodeReference.afterGetLocalNodeReferenceCounter, 1)

	if mmGetLocalNodeReference.inspectFuncGetLocalNodeReference != nil {
		mmGetLocalNodeReference.inspectFuncGetLocalNodeReference()
	}

	if mmGetLocalNodeReference.GetLocalNodeReferenceMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetLocalNodeReference.GetLocalNodeReferenceMock.defaultExpectation.Counter, 1)

		mm_results := mmGetLocalNodeReference.GetLocalNodeReferenceMock.defaultExpectation.results
		if mm_results == nil {
			mmGetLocalNodeReference.t.Fatal("No results are set for the NodeNetworkMock.GetLocalNodeReference")
		}
		return (*mm_results).h1
	}
	if mmGetLocalNodeReference.funcGetLocalNodeReference != nil {
		return mmGetLocalNodeReference.funcGetLocalNodeReference()
	}
	mmGetLocalNodeReference.t.Fatalf("Unexpected call to NodeNetworkMock.GetLocalNodeReference.")
	return
}

// GetLocalNodeReferenceAfterCounter returns a count of finished NodeNetworkMock.GetLocalNodeReference invocations
func (mmGetLocalNodeReference *NodeNetworkMock) GetLocalNodeReferenceAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLocalNodeReference.afterGetLocalNodeReferenceCounter)
}

// GetLocalNodeReferenceBeforeCounter returns a count of NodeNetworkMock.GetLocalNodeReference invocations
func (mmGetLocalNodeReference *NodeNetworkMock) GetLocalNodeReferenceBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLocalNodeReference.beforeGetLocalNodeReferenceCounter)
}

// MinimockGetLocalNodeReferenceDone returns true if the count of the GetLocalNodeReference invocations corresponds
// the number of defined expectations
func (m *NodeNetworkMock) MinimockGetLocalNodeReferenceDone() bool {
	for _, e := range m.GetLocalNodeReferenceMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLocalNodeReferenceMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeReferenceCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLocalNodeReference != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeReferenceCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetLocalNodeReferenceInspect logs each unmet expectation
func (m *NodeNetworkMock) MinimockGetLocalNodeReferenceInspect() {
	for _, e := range m.GetLocalNodeReferenceMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to NodeNetworkMock.GetLocalNodeReference")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLocalNodeReferenceMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeReferenceCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetLocalNodeReference")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLocalNodeReference != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeReferenceCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetLocalNodeReference")
	}
}

type mNodeNetworkMockGetLocalNodeRole struct {
	mock               *NodeNetworkMock
	defaultExpectation *NodeNetworkMockGetLocalNodeRoleExpectation
	expectations       []*NodeNetworkMockGetLocalNodeRoleExpectation
}

// NodeNetworkMockGetLocalNodeRoleExpectation specifies expectation struct of the NodeNetwork.GetLocalNodeRole
type NodeNetworkMockGetLocalNodeRoleExpectation struct {
	mock *NodeNetworkMock

	results *NodeNetworkMockGetLocalNodeRoleResults
	Counter uint64
}

// NodeNetworkMockGetLocalNodeRoleResults contains results of the NodeNetwork.GetLocalNodeRole
type NodeNetworkMockGetLocalNodeRoleResults struct {
	p1 member.PrimaryRole
}

// Expect sets up expected params for NodeNetwork.GetLocalNodeRole
func (mmGetLocalNodeRole *mNodeNetworkMockGetLocalNodeRole) Expect() *mNodeNetworkMockGetLocalNodeRole {
	if mmGetLocalNodeRole.mock.funcGetLocalNodeRole != nil {
		mmGetLocalNodeRole.mock.t.Fatalf("NodeNetworkMock.GetLocalNodeRole mock is already set by Set")
	}

	if mmGetLocalNodeRole.defaultExpectation == nil {
		mmGetLocalNodeRole.defaultExpectation = &NodeNetworkMockGetLocalNodeRoleExpectation{}
	}

	return mmGetLocalNodeRole
}

// Inspect accepts an inspector function that has same arguments as the NodeNetwork.GetLocalNodeRole
func (mmGetLocalNodeRole *mNodeNetworkMockGetLocalNodeRole) Inspect(f func()) *mNodeNetworkMockGetLocalNodeRole {
	if mmGetLocalNodeRole.mock.inspectFuncGetLocalNodeRole != nil {
		mmGetLocalNodeRole.mock.t.Fatalf("Inspect function is already set for NodeNetworkMock.GetLocalNodeRole")
	}

	mmGetLocalNodeRole.mock.inspectFuncGetLocalNodeRole = f

	return mmGetLocalNodeRole
}

// Return sets up results that will be returned by NodeNetwork.GetLocalNodeRole
func (mmGetLocalNodeRole *mNodeNetworkMockGetLocalNodeRole) Return(p1 member.PrimaryRole) *NodeNetworkMock {
	if mmGetLocalNodeRole.mock.funcGetLocalNodeRole != nil {
		mmGetLocalNodeRole.mock.t.Fatalf("NodeNetworkMock.GetLocalNodeRole mock is already set by Set")
	}

	if mmGetLocalNodeRole.defaultExpectation == nil {
		mmGetLocalNodeRole.defaultExpectation = &NodeNetworkMockGetLocalNodeRoleExpectation{mock: mmGetLocalNodeRole.mock}
	}
	mmGetLocalNodeRole.defaultExpectation.results = &NodeNetworkMockGetLocalNodeRoleResults{p1}
	return mmGetLocalNodeRole.mock
}

//Set uses given function f to mock the NodeNetwork.GetLocalNodeRole method
func (mmGetLocalNodeRole *mNodeNetworkMockGetLocalNodeRole) Set(f func() (p1 member.PrimaryRole)) *NodeNetworkMock {
	if mmGetLocalNodeRole.defaultExpectation != nil {
		mmGetLocalNodeRole.mock.t.Fatalf("Default expectation is already set for the NodeNetwork.GetLocalNodeRole method")
	}

	if len(mmGetLocalNodeRole.expectations) > 0 {
		mmGetLocalNodeRole.mock.t.Fatalf("Some expectations are already set for the NodeNetwork.GetLocalNodeRole method")
	}

	mmGetLocalNodeRole.mock.funcGetLocalNodeRole = f
	return mmGetLocalNodeRole.mock
}

// GetLocalNodeRole implements network.NodeNetwork
func (mmGetLocalNodeRole *NodeNetworkMock) GetLocalNodeRole() (p1 member.PrimaryRole) {
	mm_atomic.AddUint64(&mmGetLocalNodeRole.beforeGetLocalNodeRoleCounter, 1)
	defer mm_atomic.AddUint64(&mmGetLocalNodeRole.afterGetLocalNodeRoleCounter, 1)

	if mmGetLocalNodeRole.inspectFuncGetLocalNodeRole != nil {
		mmGetLocalNodeRole.inspectFuncGetLocalNodeRole()
	}

	if mmGetLocalNodeRole.GetLocalNodeRoleMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmGetLocalNodeRole.GetLocalNodeRoleMock.defaultExpectation.Counter, 1)

		mm_results := mmGetLocalNodeRole.GetLocalNodeRoleMock.defaultExpectation.results
		if mm_results == nil {
			mmGetLocalNodeRole.t.Fatal("No results are set for the NodeNetworkMock.GetLocalNodeRole")
		}
		return (*mm_results).p1
	}
	if mmGetLocalNodeRole.funcGetLocalNodeRole != nil {
		return mmGetLocalNodeRole.funcGetLocalNodeRole()
	}
	mmGetLocalNodeRole.t.Fatalf("Unexpected call to NodeNetworkMock.GetLocalNodeRole.")
	return
}

// GetLocalNodeRoleAfterCounter returns a count of finished NodeNetworkMock.GetLocalNodeRole invocations
func (mmGetLocalNodeRole *NodeNetworkMock) GetLocalNodeRoleAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLocalNodeRole.afterGetLocalNodeRoleCounter)
}

// GetLocalNodeRoleBeforeCounter returns a count of NodeNetworkMock.GetLocalNodeRole invocations
func (mmGetLocalNodeRole *NodeNetworkMock) GetLocalNodeRoleBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmGetLocalNodeRole.beforeGetLocalNodeRoleCounter)
}

// MinimockGetLocalNodeRoleDone returns true if the count of the GetLocalNodeRole invocations corresponds
// the number of defined expectations
func (m *NodeNetworkMock) MinimockGetLocalNodeRoleDone() bool {
	for _, e := range m.GetLocalNodeRoleMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLocalNodeRoleMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeRoleCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLocalNodeRole != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeRoleCounter) < 1 {
		return false
	}
	return true
}

// MinimockGetLocalNodeRoleInspect logs each unmet expectation
func (m *NodeNetworkMock) MinimockGetLocalNodeRoleInspect() {
	for _, e := range m.GetLocalNodeRoleMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to NodeNetworkMock.GetLocalNodeRole")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.GetLocalNodeRoleMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeRoleCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetLocalNodeRole")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcGetLocalNodeRole != nil && mm_atomic.LoadUint64(&m.afterGetLocalNodeRoleCounter) < 1 {
		m.t.Error("Expected call to NodeNetworkMock.GetLocalNodeRole")
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

		m.MinimockGetLatestAccessorInspect()

		m.MinimockGetLocalNodeReferenceInspect()

		m.MinimockGetLocalNodeRoleInspect()

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
		m.MinimockGetLatestAccessorDone() &&
		m.MinimockGetLocalNodeReferenceDone() &&
		m.MinimockGetLocalNodeRoleDone() &&
		m.MinimockGetOriginDone()
}