// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package s_artifact // nolint:golint

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/artifacts"
)

type ArtifactClientService interface {
	artifacts.Client
}

type ArtifactClientServiceAdapter struct {
	svc  ArtifactClientService
	exec smachine.ExecutionAdapter
}

func (a *ArtifactClientServiceAdapter) PrepareSync(
	ctx smachine.ExecutionContext,
	fn func(svc ArtifactClientService),
) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *ArtifactClientServiceAdapter) PrepareAsync(
	ctx smachine.ExecutionContext,
	fn func(svc ArtifactClientService) smachine.AsyncResultFunc,
) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *ArtifactClientServiceAdapter) PrepareNotify(
	ctx smachine.ExecutionContext,
	fn func(svc ArtifactClientService),
) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) {
		fn(a.svc)
	})
}

type artifactClientService struct {
	artifacts.Client
}

func CreateArtifactClientService(client artifacts.Client) *ArtifactClientServiceAdapter {
	ctx := context.Background()
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 16)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &ArtifactClientServiceAdapter{
		svc: artifactClientService{
			Client: client,
		},
		exec: smachine.NewExecutionAdapter("ArtifactClientService", ae),
	}
}
