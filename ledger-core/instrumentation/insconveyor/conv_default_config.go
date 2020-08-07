// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

func DefaultConfig() conveyor.PulseConveyorConfig {
	return DefaultConfigWithOverrides(ConfigOverrides{})
}

func DefaultConfigNoEventless() conveyor.PulseConveyorConfig {
	return DefaultConfigWithOverrides(ConfigOverrides{ EventlessSleep: -1})
}

func DefaultConfigWithOverrides(p ConfigOverrides) conveyor.PulseConveyorConfig {
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
	if p.MachineLogger != nil {
		machineConfig.SlotMachineLogger = p.MachineLogger
	}

	switch {
	case p.BoostNewSlotDuration > 0:
		machineConfig.BoostNewSlotDuration = p.BoostNewSlotDuration
	case p.BoostNewSlotDuration == 0:
		machineConfig.BoostNewSlotDuration = 0 * time.Millisecond
	}

	switch {
	case p.EventlessSleep == 0:
		p.EventlessSleep = 100 * time.Millisecond
	case p.EventlessSleep < 0:
		p.EventlessSleep = 0
	}

	return conveyor.PulseConveyorConfig{
		ConveyorMachineConfig: conveyorConfig,
		SlotMachineConfig:     machineConfig,
		EventlessSleep:        p.EventlessSleep,
		MinCachePulseAge:      100,
		MaxPastPulseAge:       1000,
	}
}
