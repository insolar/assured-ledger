package testutils

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine/machinetype"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

type descriptorPair struct {
	class descriptor.Class
	code  descriptor.Code
}

type DescriptorCacheMockWrapper struct {
	childMock      *descriptor.CacheMock
	T              *testing.T
	Classes        map[reference.Global]descriptorPair
	IntenselyPanic bool
}

func NewDescriptorsCacheMockWrapper(mc minimock.Tester) *DescriptorCacheMockWrapper {
	mock := DescriptorCacheMockWrapper{
		childMock:      descriptor.NewCacheMock(mc),
		Classes:        make(map[reference.Global]descriptorPair),
		IntenselyPanic: false,
	}

	return &mock
}

func (w *DescriptorCacheMockWrapper) byClassRefImpl(
	_ context.Context,
	classRef reference.Global,
) (
	descriptor.Class,
	descriptor.Code,
	error,
) {
	if pair, ok := w.Classes[classRef]; ok {
		return pair.class, pair.code, nil
	}

	if w.IntenselyPanic {
		panic(throw.E("object not found", struct{ id string }{id: classRef.String()}))
	}

	return nil, nil, errors.New("object not found")
}

func (w *DescriptorCacheMockWrapper) AddClassCodeDescriptor(
	head reference.Global,
	state reference.Local,
	code reference.Global,
) {
	if _, ok := w.Classes[head]; ok {
		panic("already exists")
	}

	w.Classes[head] = descriptorPair{
		class: descriptor.NewClass(head, state, code),
		code:  descriptor.NewCode(gen.UniqueGlobalRef().AsBytes(), machinetype.Builtin, code),
	}

	w.childMock.ByClassRefMock.Set(w.byClassRefImpl)
}

func (w *DescriptorCacheMockWrapper) Mock() *descriptor.CacheMock {
	return w.childMock
}
