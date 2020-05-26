package object

import (
	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type catalogMockWrapperEntry struct {
	accessor    SharedStateAccessor
	wasAccessed bool
}

type CatalogMockWrapper struct {
	t         minimock.Tester
	objects   map[reference.Global]*catalogMockWrapperEntry
	childMock *CatalogMock
}

func NewCatalogMockWrapper(t minimock.Tester) *CatalogMockWrapper {
	mock := CatalogMockWrapper{
		t:         t,
		objects:   make(map[reference.Global]*catalogMockWrapperEntry),
		childMock: NewCatalogMock(t),
	}
	mock.childMock.GetOrCreateMock.Set(mock.mockGetOrCreate)
	return &mock
}

func (m *CatalogMockWrapper) Mock() *CatalogMock {
	return m.childMock
}

func (m *CatalogMockWrapper) AddObject(objectRef reference.Global, objectAccessor SharedStateAccessor) *CatalogMockWrapper {
	m.objects[objectRef] = &catalogMockWrapperEntry{accessor: objectAccessor, wasAccessed: false}
	return m
}

func (m *CatalogMockWrapper) mockGetOrCreate(_ smachine.ExecutionContext, ref reference.Global) SharedStateAccessor {
	entry, ok := m.objects[ref]
	if !ok {
		panic(throw.Unsupported())
	}
	entry.wasAccessed = true
	return entry.accessor
}
