// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

var _ descriptor.Cache = &descriptorsCache{}

type descriptorsCache struct {
	callbacks []descriptor.CacheCallbackType

	codeCache  cache
	protoCache cache
}

func NewDescriptorsCache() descriptor.Cache {
	return &descriptorsCache{
		callbacks: make([]descriptor.CacheCallbackType, 0),

		codeCache:  newSingleFlightCache(),
		protoCache: newSingleFlightCache(),
	}
}

func (c *descriptorsCache) RegisterCallback(cb descriptor.CacheCallbackType) {
	c.callbacks = append(c.callbacks, cb)
}

func (c *descriptorsCache) ByPrototypeRef(
	ctx context.Context, protoRef reference.Global,
) (
	descriptor.PrototypeDescriptor, descriptor.CodeDescriptor, error,
) {
	protoDesc, err := c.GetPrototype(ctx, protoRef)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get prototype descriptor")
	}

	codeRef := protoDesc.Code()
	codeDesc, err := c.GetCode(ctx, *codeRef)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get code descriptor")
	}

	return protoDesc, codeDesc, nil
}

func (c *descriptorsCache) ByObjectDescriptor(
	ctx context.Context, obj descriptor.ObjectDescriptor,
) (
	descriptor.PrototypeDescriptor, descriptor.CodeDescriptor, error,
) {
	protoRef, err := obj.Prototype()
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't get prototype reference")
	}

	if protoRef == nil {
		return nil, nil, errors.New("Empty prototype")
	}

	return c.ByPrototypeRef(ctx, *protoRef)
}

func (c *descriptorsCache) GetPrototype(
	ctx context.Context, ref reference.Global,
) (
	descriptor.PrototypeDescriptor, error,
) {
	rawResult, err := c.protoCache.get(ref, func() (interface{}, error) {
		for _, cb := range c.callbacks {
			object, err := cb(ref)
			if object != nil || err != nil {
				return object, err
			}
		}
		return nil, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "couldn't get prototype")
	} else if rawResult == nil {
		return nil, errors.Errorf("failed to find prototype descriptor %s", ref.String())
	}

	result, ok := rawResult.(descriptor.PrototypeDescriptor)
	if !ok {
		return nil, errors.Errorf("unexpected type %T, expected PrototypeDescriptor", rawResult)
	}

	return result, nil
}

func (c *descriptorsCache) GetCode(
	ctx context.Context, ref reference.Global,
) (
	descriptor.CodeDescriptor, error,
) {
	rawResult, err := c.codeCache.get(ref, func() (interface{}, error) {
		for _, cb := range c.callbacks {
			object, err := cb(ref)
			if object != nil || err != nil {
				return object, err
			}
		}
		return nil, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "couldn't get code")
	} else if rawResult == nil {
		return nil, errors.Errorf("failed to find code descriptor %s", ref.String())
	}

	result, ok := rawResult.(descriptor.CodeDescriptor)
	if !ok {
		return nil, errors.Errorf("unexpected type %T, expected CodeDescriptor", rawResult)
	}

	return result, nil
}

type cache interface {
	get(ref reference.Global, getter func() (val interface{}, err error)) (val interface{}, err error)
}

type cacheEntry struct {
	mu    sync.Mutex
	value interface{}
}

type singleFlightCache struct {
	mu sync.Mutex
	m  map[reference.Global]*cacheEntry
}

func newSingleFlightCache() cache {
	return &singleFlightCache{
		m: make(map[reference.Global]*cacheEntry),
	}
}

func (c *singleFlightCache) getEntry(ref reference.Global) *cacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.m[ref]; !ok {
		c.m[ref] = &cacheEntry{}
	}
	return c.m[ref]
}

func (c *singleFlightCache) get(
	ref reference.Global,
	getter func() (value interface{}, err error),
) (
	interface{}, error,
) {
	e := c.getEntry(ref)

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.value != nil {
		return e.value, nil
	}

	val, err := getter()
	if err != nil {
		return val, err
	}

	e.value = val
	return e.value, nil
}
