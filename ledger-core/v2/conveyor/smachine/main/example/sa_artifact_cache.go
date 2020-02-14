// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type ArtifactCacheId string

type ArtifactCacheService interface {
	Set(objectID insolar.ID, object []byte) ArtifactCacheId
	SetRandomID(object []byte) (ArtifactCacheId, error)
	Get(id ArtifactCacheId) ([]byte, bool)
}

type ArtifactCacheServiceAdapter struct {
	svc  ArtifactCacheService
	exec smachine.ExecutionAdapter
}

func (a *ArtifactCacheServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc ArtifactCacheService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ArtifactCacheServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc ArtifactCacheService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func CreateArtifactCacheService() *ArtifactCacheServiceAdapter {
	ctx := context.Background()
	ae, ch := smachine.NewCallChannelExecutor(ctx, 0, false, 5)

	smachine.StartChannelWorker(ctx, ch, nil)

	return &ArtifactCacheServiceAdapter{
		svc: &unlimitedArtifactCacheService{
			cache: map[ArtifactCacheId][]byte{},
		},
		exec: smachine.NewExecutionAdapter("ArtifactCache", ae),
	}
}

type unlimitedArtifactCacheService struct {
	lock  sync.RWMutex
	cache map[ArtifactCacheId][]byte
}

func (a *unlimitedArtifactCacheService) Set(objectID insolar.ID, object []byte) ArtifactCacheId {
	a.lock.Lock()
	defer a.lock.Unlock()

	cacheID := ArtifactCacheId(objectID.String())

	a.cache[cacheID] = object

	return cacheID
}

func (a *unlimitedArtifactCacheService) SetRandomID(object []byte) (ArtifactCacheId, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	rawCacheID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Wrap(err, "failed to get id for request")
	}
	cacheID := ArtifactCacheId(rawCacheID.String())

	a.cache[cacheID] = object

	return cacheID, nil
}

func (a *unlimitedArtifactCacheService) Get(id ArtifactCacheId) ([]byte, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	rv, ok := a.cache[id]
	return rv, ok
}
