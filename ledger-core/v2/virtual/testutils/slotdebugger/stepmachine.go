// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotdebugger

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

type VirtualStepController struct {
	*slotdebugger.StepController

	RunnerDescriptorCache *testutils.DescriptorCacheMockWrapper
	MachineManager        machine.Manager
	MessageSender         *messagesender.ServiceMockWrapper
}

func New(ctx context.Context, t *testing.T, suppressLogError bool) *VirtualStepController {
	stepController := slotdebugger.New(ctx, t, suppressLogError)

	w := &VirtualStepController{
		StepController: stepController,
	}

	return w
}

func (c *VirtualStepController) PrepareRunner(ctx context.Context, mc *minimock.Controller) {
	c.RunnerDescriptorCache = testutils.NewDescriptorsCacheMockWrapper(mc)
	c.MachineManager = machine.NewManager()

	runnerService := runner.NewService()
	runnerService.Manager = c.MachineManager
	runnerService.Cache = c.RunnerDescriptorCache.Mock()

	runnerAdapter := runner.CreateRunnerService(ctx, runnerService)
	c.SlotMachine.AddDependency(runnerAdapter)
}
