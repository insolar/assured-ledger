// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"context"
	"sync"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
)

var _ descriptor.DescriptorsCache = &descriptorsCache{}

type descriptorsCache struct {
	codeCache  cache
	protoCache cache
}

func NewDescriptorsCache() descriptor.DescriptorsCache {
	return &descriptorsCache{
		codeCache:  newSingleFlightCache(),
		protoCache: newSingleFlightCache(),
	}
}

func (c *descriptorsCache) ByPrototypeRef(
	ctx context.Context, protoRef insolar.Reference,
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
	ctx context.Context, ref insolar.Reference,
) (
	descriptor.PrototypeDescriptor, error,
) {
	res, err := c.protoCache.get(ref, func() (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get object")
	}

	return res.(descriptor.PrototypeDescriptor), nil
}

func (c *descriptorsCache) GetCode(
	ctx context.Context, ref insolar.Reference,
) (
	descriptor.CodeDescriptor, error,
) {
	res, err := c.codeCache.get(ref, func() (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get code")
	}
	return res.(descriptor.CodeDescriptor), nil
}

type cache interface {
	get(ref insolar.Reference, getter func() (val interface{}, err error)) (val interface{}, err error)
}

type cacheEntry struct {
	mu    sync.Mutex
	value interface{}
}

type singleFlightCache struct {
	mu sync.Mutex
	m  map[insolar.Reference]*cacheEntry
}

func newSingleFlightCache() cache {
	return &singleFlightCache{
		m: make(map[insolar.Reference]*cacheEntry),
	}
}

func (c *singleFlightCache) getEntry(ref insolar.Reference) *cacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.m[ref]; !ok {
		c.m[ref] = &cacheEntry{}
	}
	return c.m[ref]
}

func (c *singleFlightCache) get(
	ref insolar.Reference,
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
