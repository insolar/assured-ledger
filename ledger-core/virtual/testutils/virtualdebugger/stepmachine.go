package virtualdebugger

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/log/logcommon"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/machine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
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

	{
		keyProcessor := platformpolicy.NewKeyProcessor()
		pk, err := keyProcessor.GeneratePrivateKey()
		if err != nil {
			panic(throw.W(err, "failed to generate node PK"))
		}
		keyStore := keystore.NewInplaceKeyStore(pk)

		platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
		platformScheme := legacyadapter.New(platformCryptographyScheme, keyProcessor, keyStore)

		w.SlotMachine.AddInterfaceDependency(&platformScheme)
		// var rrb vnlmn.RecordReferenceBuilder
		// rrb = vnlmn.NewRecordReferenceBuilder(platformScheme.RecordScheme(), w.GenerateGlobal())
		// w.SlotMachine.AddInterfaceDependency(&rrb)
	}

	return w
}

func New(ctx context.Context, t *testing.T) *VirtualStepController {
	return NewWithErrorFilter(ctx, t, nil)
}

// deprecated
func NewWithIgnoreAllError(ctx context.Context, t *testing.T) *VirtualStepController {
	return NewWithErrorFilter(ctx, t, func(s string) bool { return false })

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
