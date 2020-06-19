// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotdebugger

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

type VirtualStepController struct {
	*slotdebugger.StepController

	RunnerMock            *logicless.ServiceMock
	RunnerDescriptorCache *testutils.DescriptorCacheMockWrapper
	MachineManager        machine.Manager
}

func New(ctx context.Context, t *testing.T, suppressLogError bool) *VirtualStepController {
	stepController := slotdebugger.New(ctx, t, suppressLogError)

	w := &VirtualStepController{
		StepController: stepController,
	}

	return w
}

func (c *VirtualStepController) PrepareRunner(ctx context.Context, mc minimock.Tester) {
	c.RunnerDescriptorCache = testutils.NewDescriptorsCacheMockWrapper(mc)
	c.MachineManager = machine.NewManager()

	runnerService := runner.NewService()
	runnerService.Manager = c.MachineManager
	runnerService.Cache = c.RunnerDescriptorCache.Mock()

	runnerAdapter := runnerService.CreateAdapter(ctx)
	c.SlotMachine.AddInterfaceDependency(&runnerAdapter)
}

func (c *VirtualStepController) PrepareMockedRunner(ctx context.Context, mc minimock.Tester) {
	c.RunnerMock = logicless.NewServiceMock(ctx, mc, nil)

	runnerAdapter := c.RunnerMock.CreateAdapter(ctx)
	c.SlotMachine.AddInterfaceDependency(&runnerAdapter)
}

func (c VirtualStepController) GenerateLocal() reference.Local {
	return gen.UniqueIDWithPulse(c.PulseSlot.CurrentPulseNumber())
}

func (c VirtualStepController) GenerateGlobal() reference.Global {
	return reference.NewSelf(c.GenerateLocal())
}
