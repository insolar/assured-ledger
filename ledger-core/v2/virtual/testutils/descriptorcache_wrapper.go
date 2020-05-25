// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type descriptorPair struct {
	proto descriptor.Prototype
	code  descriptor.Code
}

type DescriptorCacheMockWrapper struct {
	childMock      *descriptor.CacheMock
	T              *testing.T
	Prototypes     map[reference.Global]descriptorPair
	IntenselyPanic bool
}

func NewDescriptorsCacheMockWrapper(mc *minimock.Controller) *DescriptorCacheMockWrapper {
	mock := DescriptorCacheMockWrapper{
		childMock:      descriptor.NewCacheMock(mc),
		Prototypes:     make(map[reference.Global]descriptorPair),
		IntenselyPanic: false,
	}

	return &mock
}

func (w *DescriptorCacheMockWrapper) byPrototypeRefImpl(
	_ context.Context,
	protoRef reference.Global,
) (
	descriptor.Prototype,
	descriptor.Code,
	error,
) {
	if pair, ok := w.Prototypes[protoRef]; ok {
		return pair.proto, pair.code, nil
	}

	if w.IntenselyPanic {
		panic(throw.E("object not found", struct{ id string }{id: protoRef.String()}))
	}

	return nil, nil, errors.New("object not found")
}

func (w *DescriptorCacheMockWrapper) AddPrototypeCodeDescriptor(
	head reference.Global,
	state reference.Local,
	code reference.Global,
) {
	if _, ok := w.Prototypes[head]; ok {
		panic("already exists")
	}

	w.Prototypes[head] = descriptorPair{
		proto: descriptor.NewPrototype(head, state, code),
		code:  descriptor.NewCode(gen.UniqueReference().AsBytes(), machine.Builtin, code),
	}

	w.childMock.ByPrototypeRefMock.Set(w.byPrototypeRefImpl)
}

func (w *DescriptorCacheMockWrapper) Mock() *descriptor.CacheMock {
	return w.childMock
}
