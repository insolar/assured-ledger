package insconveyor

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type ConfigOverrides struct {
	BoostNewSlotDuration time.Duration
	MachineLogger  smachine.SlotMachineLogger
	EventlessSleep time.Duration
}

func DefaultConfigNoEventless() conveyor.PulseConveyorConfig {
	cfg := DefaultConfig()
	cfg.EventlessSleep = 0
	return cfg
}

func DefaultConfig() conveyor.PulseConveyorConfig {
	conveyorConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: ConveyorLoggerFactory{},
		SlotAliasRegistry: &conveyor.GlobalAliases{},
		LogAdapterCalls:   true,
	}

	machineConfig := conveyorConfig

	return conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: conveyorConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        500 * time.Millisecond,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}
}

func ApplyConfigOverrides(cfg *conveyor.PulseConveyorConfig, p ConfigOverrides) {
	if p.MachineLogger != nil {
		cfg.SlotMachineConfig.SlotMachineLogger = p.MachineLogger
	}

	switch {
	case p.BoostNewSlotDuration > 0:
		cfg.SlotMachineConfig.BoostNewSlotDuration = p.BoostNewSlotDuration
	case p.BoostNewSlotDuration < 0:
		cfg.SlotMachineConfig.BoostNewSlotDuration = 0
	}

	switch {
	case p.EventlessSleep > 0:
		cfg.EventlessSleep = p.EventlessSleep
	case p.EventlessSleep < 0:
		cfg.EventlessSleep = 0
	}
}
