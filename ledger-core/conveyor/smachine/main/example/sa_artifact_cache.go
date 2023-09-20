package example

import (
	"context"
	"sync"

	uuid "github.com/satori/go.uuid"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type ArtifactCacheID string

type ArtifactCacheService interface {
	Set(objectID reference.Local, object []byte) ArtifactCacheID
	SetRandomID(object []byte) (ArtifactCacheID, error)
	Get(id ArtifactCacheID) ([]byte, bool)
}

type ArtifactCacheServiceAdapter struct {
	svc  ArtifactCacheService
	exec smachine.ExecutionAdapter
}

func (a *ArtifactCacheServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc ArtifactCacheService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ArtifactCacheServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc ArtifactCacheService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func CreateArtifactCacheService() *ArtifactCacheServiceAdapter {
	ctx := context.Background()
	ae, ch := smadapter.NewCallChannelExecutor(ctx, 0, false, 5)

	smachine.StartChannelWorker(ctx, ch, nil)

	return &ArtifactCacheServiceAdapter{
		svc: &unlimitedArtifactCacheService{
			cache: map[ArtifactCacheID][]byte{},
		},
		exec: smachine.NewExecutionAdapter("ArtifactCache", ae),
	}
}

type unlimitedArtifactCacheService struct {
	lock  sync.RWMutex
	cache map[ArtifactCacheID][]byte
}

func (a *unlimitedArtifactCacheService) Set(objectID reference.Local, object []byte) ArtifactCacheID {
	a.lock.Lock()
	defer a.lock.Unlock()

	cacheID := ArtifactCacheID(objectID.String())

	a.cache[cacheID] = object

	return cacheID
}

func (a *unlimitedArtifactCacheService) SetRandomID(object []byte) (ArtifactCacheID, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	rawCacheID, err := uuid.NewV4()
	if err != nil {
		return "", errors.W(err, "failed to get id for request")
	}
	cacheID := ArtifactCacheID(rawCacheID.String())

	a.cache[cacheID] = object

	return cacheID, nil
}

func (a *unlimitedArtifactCacheService) Get(id ArtifactCacheID) ([]byte, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()

	rv, ok := a.cache[id]
	return rv, ok
}
