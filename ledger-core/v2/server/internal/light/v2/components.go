package v2

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/smachines"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
)

type components struct {
}

func newComponents() *components {
	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: statemachine.ConveyorLoggerFactory{},
	}

	conv := conveyor.NewPulseConveyor(context.Background(), conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: machineConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        100 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}, smachines.CommonFactory, nil)

	worker := smachines.NewWorker()
	worker.AttachTo(conv)

	disp := smachines.NewDispatcher(conv)
	_ = disp

	conv.AddDependency(smachines.NewHashingAdapter())
	conv.AddDependency(smachines.NewSyncAdapter())
	conv.AddDependency(platformpolicy.NewPlatformCryptographyScheme())
	conv.AddDependency(store.NewRecordStore())

	return &components{}
}
