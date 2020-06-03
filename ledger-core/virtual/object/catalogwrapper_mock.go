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

type CatalogMockAccessType int

const (
	CatalogMockAccessGetOrCreate CatalogMockAccessType = iota
	CatalogMockAccessTryGet
)

func NewCatalogMockWrapper(t minimock.Tester) *CatalogMockWrapper {
	return &CatalogMockWrapper{
		t:         t,
		objects:   make(map[reference.Global]*catalogMockWrapperEntry),
		childMock: NewCatalogMock(t),
	}
}

func (m *CatalogMockWrapper) AllowAccessMode(mode CatalogMockAccessType) {
	switch mode {
	case CatalogMockAccessGetOrCreate:
		m.childMock.GetOrCreateMock.Set(m.mockGetOrCreate)
	case CatalogMockAccessTryGet:
		m.childMock.TryGetMock.Set(m.mockTryGet)
	default:
		m.t.Fatal(throw.NotImplemented())
	}
}

func (m *CatalogMockWrapper) CheckDone() error {
	for _, entry := range m.objects {
		if !entry.wasAccessed {
			return throw.New("not all object from catalog was used")
		}
	}
	return nil
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

func (m *CatalogMockWrapper) mockTryGet(_ smachine.ExecutionContext, ref reference.Global) (SharedStateAccessor, bool) {
	entry, ok := m.objects[ref]
	if !ok {
		return SharedStateAccessor{}, false
	}
	entry.wasAccessed = true
	return entry.accessor, true
}
