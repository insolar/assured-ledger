// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package virtualdebugger

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

type VirtualStepController struct {
	*slotdebugger.StepController

	RunnerMock            *logicless.ServiceMock
	RunnerDescriptorCache *testutils.DescriptorCacheMockWrapper
	MachineManager        machine.Manager
	MemoryCache           memoryCacheAdapter.MemoryCache
}

func NewWithErrorFilter(ctx context.Context, t *testing.T, filterFn logcommon.ErrorFilterFunc) *VirtualStepController {
	stepController := slotdebugger.NewWithErrorFilter(ctx, t, filterFn)
	w := &VirtualStepController{
		StepController: stepController,
	}

	return w
}

func New(ctx context.Context, t *testing.T) *VirtualStepController {
	return NewWithErrorFilter(ctx, t, nil)
}

// deprecated
func NewWithIgnoreAllError(ctx context.Context, t *testing.T) *VirtualStepController {
	stepController := slotdebugger.NewWithIgnoreAllErrors(ctx, t)

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
	return gen.UniqueLocalRefWithPulse(c.PulseSlot.CurrentPulseNumber())
}

func (c VirtualStepController) GenerateGlobal() reference.Global {
	return reference.NewSelf(c.GenerateLocal())
}
