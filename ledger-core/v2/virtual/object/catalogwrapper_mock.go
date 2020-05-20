package object

import (
	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type catalogWrapperMockEntry struct {
	accessor    SharedStateAccessor
	wasAccessed bool
}

type CatalogWrapperMock struct {
	t         minimock.Tester
	objects   map[reference.Global]*catalogWrapperMockEntry
	childMock *CatalogMock
}

func NewCatalogWrapperMock(t minimock.Tester) *CatalogWrapperMock {
	mock := CatalogWrapperMock{
		t:         t,
		objects:   make(map[reference.Global]*catalogWrapperMockEntry),
		childMock: NewCatalogMock(t),
	}
	mock.childMock.GetOrCreateMock.Set(mock.mockGetOrCreate)
	return &mock
}

func (m *CatalogWrapperMock) Mock() *CatalogMock {
	return m.childMock
}

func (m *CatalogWrapperMock) AddObject(objectRef reference.Global, objectAccessor SharedStateAccessor) *CatalogWrapperMock {
	m.objects[objectRef] = &catalogWrapperMockEntry{accessor: objectAccessor, wasAccessed: false}
	return m
}

func (m *CatalogWrapperMock) mockGetOrCreate(_ smachine.ExecutionContext, ref reference.Global) SharedStateAccessor {
	entry, ok := m.objects[ref]
	if !ok {
		panic(throw.Unsupported())
	}
	entry.wasAccessed = true
	return entry.accessor
}
