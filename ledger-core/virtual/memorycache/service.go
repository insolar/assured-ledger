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
	Get(ctx context.Context, objectDescriptorRef reference.Global, stateRef reference.Global) (descriptor.Object, error)
	Set(ctx context.Context, objectDescriptorRef reference.Global, stateRef reference.Global, objectDescriptor descriptor.Object) error
}

type DefaultService struct {
	memoryCache *LRUMemoryCache
}

func (s *DefaultService) Get(ctx context.Context, objectDescriptorRef reference.Global, stateRef reference.Global) (descriptor.Object, error) {
	key := Key{
		objectDescriptorRef: objectDescriptorRef,
		stateRef:            stateRef,
	}
	object, found := s.memoryCache.Get(key)
	if !found {
		return nil, throw.New("key not found")
	}
	return object, nil
}

func (s *DefaultService) Set(ctx context.Context, objectDescriptorRef reference.Global, stateRef reference.Global, objectDescriptor descriptor.Object) error {
	key := Key{
		objectDescriptorRef: objectDescriptorRef,
		stateRef:            objectDescriptorRef, // must be stateRef
	}
	added := s.memoryCache.Replace(key, objectDescriptor)
	if !added {
		inslogger.FromContext(ctx).Debug("key already exists in cache, value updated")
	}
	return nil
}

func NewDefaultService() *DefaultService {
	strategy := cacheStrategy{
		pgSize:   10,
		maxTotal: 100,
		trimEach: true,
	}
	return &DefaultService{
		memoryCache: NewMemoryCache(strategy),
	}
}
