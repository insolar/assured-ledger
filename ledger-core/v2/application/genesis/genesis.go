// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package genesis

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/application"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	insolarPulse "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
)

// BaseRecord provides methods for genesis base record manipulation.
type BaseRecord struct {
	DB            store.DB
	PulseAppender insolarPulse.Appender
	PulseAccessor insolarPulse.Accessor
}

// Key is genesis key.
type Key struct{}

func (Key) ID() []byte {
	return insolar.GenesisPulse.PulseNumber.Bytes()
}

func (Key) Scope() store.Scope {
	return store.ScopeGenesis
}

// IsGenesisRequired checks if genesis record already exists.
func (br *BaseRecord) IsGenesisRequired(ctx context.Context) (bool, error) {
	b, err := br.DB.Get(Key{})
	if err != nil {
		if err == store.ErrNotFound {
			return true, nil
		}
		return false, errors.Wrap(err, "genesis record fetch failed")
	}

	if len(b) == 0 {
		return false, errors.New("genesis record is empty (genesis hasn't properly finished)")
	}
	return false, nil
}

// Create creates new base genesis record if needed.
func (br *BaseRecord) Create(ctx context.Context) error {
	inslog := inslogger.FromContext(ctx)
	inslog.Info("start storage bootstrap")

	err := br.PulseAppender.Append(
		ctx,
		insolar.Pulse{
			PulseNumber: insolar.GenesisPulse.PulseNumber,
			Entropy:     insolar.GenesisPulse.Entropy,
		},
	)
	if err != nil {
		return errors.Wrap(err, "fail to set genesis pulse")
	}

	lastPulse, err := br.PulseAccessor.Latest(ctx)
	if err != nil {
		return errors.Wrap(err, "fail to get last pulse")
	}
	if lastPulse.PulseNumber != insolar.GenesisPulse.PulseNumber {
		return fmt.Errorf(
			"last pulse number %v is not equal to genesis special value %v",
			lastPulse.PulseNumber,
			insolar.GenesisPulse.PulseNumber,
		)
	}

	return br.DB.Set(Key{}, nil)
}

// Done saves genesis value. Should be called when all genesis steps finished properly.
func (br *BaseRecord) Done(ctx context.Context) error {
	return br.DB.Set(Key{}, application.GenesisRecord.Ref().Bytes())
}

// Genesis holds data and objects required for genesis on heavy node.
type Genesis struct {
	BaseRecord *BaseRecord

	PluginsDir string
}

// Start implements components.Starter.
func (g *Genesis) Start(ctx context.Context) error {
	inslog := inslogger.FromContext(ctx)

	isRequired, err := g.BaseRecord.IsGenesisRequired(ctx)
	inslogger.FromContext(ctx).Infof("[genesis] required=%v", isRequired)
	if err != nil {
		panic(err.Error())
	}

	if !isRequired {
		inslog.Info("[genesis] base genesis record exists, skip genesis")
		return nil
	}

	inslogger.FromContext(ctx).Info("[genesis] start...")

	inslog.Info("[genesis] create genesis record")
	err = g.BaseRecord.Create(ctx)
	if err != nil {
		return err
	}

	inslog.Info("[genesis] finalize genesis record")
	err = g.BaseRecord.Done(ctx)
	if err != nil {
		panic(fmt.Sprintf("[genesis] finalize genesis record failed: %v", err))
	}

	return nil
}
