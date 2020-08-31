// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package memorycache

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/virtual/memorycache.Service -o ./ -s _mock.go -g
type Service interface {
	Get(ctx context.Context, objectReference reference.Global) (descriptor.Object, error)
	Set(ctx context.Context, objectReference reference.Global, objectDescriptor descriptor.Object) error
}

type DefaultService struct {
	memoryCache *LRUMemoryCache
}

func (s *DefaultService) Get(ctx context.Context, objectReference reference.Global) (descriptor.Object, error) {
	object, found := s.memoryCache.Get(objectReference)
	if !found {
		return nil, throw.New("key not found")
	}
	return object, nil
}

func (s *DefaultService) Set(ctx context.Context, objectReference reference.Global, objectDescriptor descriptor.Object) error {
	added := s.memoryCache.Replace(objectReference, objectDescriptor)
	if !added {
		inslogger.FromContext(ctx).Debug("key already exists in cache, value updated")
	}
	return nil
}

func NewDefaultService() *DefaultService {
	return &DefaultService{
		memoryCache: NewMemoryCache(),
	}
}
