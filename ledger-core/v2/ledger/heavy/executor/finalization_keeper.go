// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

// FinalizationKeeper check how far from each other last finalized pulse and current one
// and if distance is more than limit it stops network
type FinalizationKeeper interface {
	OnPulse(ctx context.Context, current insolar.PulseNumber) error
}

type FinalizationKeeperDefault struct {
	jetKeeper       JetKeeper
	limit           int
	pulseCalculator pulse.Calculator
}

func NewFinalizationKeeperDefault(jk JetKeeper, pc pulse.Calculator, limit int) *FinalizationKeeperDefault {
	return &FinalizationKeeperDefault{
		jetKeeper:       jk,
		limit:           limit,
		pulseCalculator: pc,
	}
}

func (f *FinalizationKeeperDefault) OnPulse(ctx context.Context, current insolar.PulseNumber) error {
	logger := inslogger.FromContext(ctx).WithFields(map[string]interface{}{
		"new_pulse": current,
	})

	logger.Infof("FinalizationKeeper.OnPulse about to call pulseCalculator.Backwards")

	bottomLevel, err := f.pulseCalculator.Backwards(ctx, current, f.limit)
	if err != nil {
		if err == pulse.ErrNotFound {
			logger.Debug("finalizationKeeper: possibly we started not so long ago. Do nothing. Current pulse: ", current)
			return nil
		}
		return errors.Wrap(err, "Can't get old pulse")
	}

	logger.Infof("FinalizationKeeper.OnPulse returned from pulseCalculator.Backwards, about to call jetKeeper.TopSyncPulse")

	lastConfirmedPulse := f.jetKeeper.TopSyncPulse()

	logger.Infof("FinalizationKeeper.OnPulse returned from jetKeeper.TopSyncPulse. Last confirmed pulse: %d, current: %d, limit: %d, bottom level: %d",
		lastConfirmedPulse, current, f.limit, bottomLevel.PulseNumber)

	if current < lastConfirmedPulse {
		return errors.New(fmt.Sprintf("Current pulse ( %d ) is less than last confirmed ( %d )", current, lastConfirmedPulse))
	}

	if lastConfirmedPulse <= bottomLevel.PulseNumber {
		logger.Panicf("last finalized pulse falls behind too much. Stop node. bottomLevel.PulseNumber: %d, last confirmed: %d", bottomLevel.PulseNumber, lastConfirmedPulse)
	}

	logger.Infof("FinalizationKeeper: everything is ok. Current pulse: %d, last confirmed: %d, limit: %d", current, lastConfirmedPulse, f.limit)

	return nil
}
