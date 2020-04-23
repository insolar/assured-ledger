// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/log"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

type descriptorPair struct {
	proto descriptor.PrototypeDescriptor
	code  descriptor.CodeDescriptor
}

type DescriptorCacheMockWrapper struct {
	*descriptor.CacheMock
	T              *testing.T
	Prototypes     map[reference.Global]descriptorPair
	IntenselyPanic bool
}

func NewDescriptorsCacheMockWrapper(t *testing.T) *DescriptorCacheMockWrapper {
	mock := DescriptorCacheMockWrapper{
		T:              t,
		CacheMock:      descriptor.NewCacheMock(t),
		Prototypes:     make(map[reference.Global]descriptorPair),
		IntenselyPanic: false,
	}

	mock.CacheMock.ByPrototypeRefMock.Set(mock.byPrototypeRefImpl)

	return &mock
}

type errPrototypeNotFound struct {
	*log.Msg `txt:"object not found"`
	id       reference.Global
}

func (w *DescriptorCacheMockWrapper) byPrototypeRefImpl(
	_ context.Context,
	protoRef reference.Global,
) (
	descriptor.PrototypeDescriptor,
	descriptor.CodeDescriptor,
	error,
) {
	if pair, ok := w.Prototypes[protoRef]; ok {
		return pair.proto, pair.code, nil
	}

	if w.IntenselyPanic {
		panic(errPrototypeNotFound{id: protoRef})
	}

	return nil, nil, errors.New("object not found")
}

// nolint:unused
func (w *DescriptorCacheMockWrapper) addPrototypeCodeDescriptor(
	head reference.Global,
	state reference.Local,
	code reference.Global,
) {
	if _, ok := w.Prototypes[head]; ok {
		panic("already exists")
	}

	w.Prototypes[head] = descriptorPair{
		proto: descriptor.NewPrototypeDescriptor(head, state, code),
		code:  descriptor.NewCodeDescriptor(nil, insolar.MachineTypeBuiltin, code),
	}
}
