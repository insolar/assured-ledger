// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type ArtifactClientService interface {
	GetLatestValidatedStateAndCode() (state, code ArtifactBinary)
}

type ArtifactBinary interface {
	GetReference() reference.Global
	GetCacheId() ArtifactCacheID
}

type ArtifactClientServiceAdapter struct {
	svc  ArtifactClientService
	exec smachine.ExecutionAdapter
}

func (a *ArtifactClientServiceAdapter) PrepareSync(ctx smachine.ExecutionContext, fn func(svc ArtifactClientService)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ArtifactClientServiceAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc ArtifactClientService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func CreateArtifactClientService() *ArtifactClientServiceAdapter {
	ctx := context.Background()
	ae, ch := smadapter.NewCallChannelExecutor(ctx, 0, false, 5)
	ea := smachine.NewExecutionAdapter("ServiceA", ae)

	smachine.StartChannelWorker(ctx, ch, nil)
	return &ArtifactClientServiceAdapter{&artifactClientService{}, ea}
}

var _ ArtifactClientService = &artifactClientService{}

type artifactClientService struct {
}

func (*artifactClientService) GetLatestValidatedStateAndCode() (state, code ArtifactBinary) {
	panic("implement me")
}
