// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

func AppFactory(_ context.Context, cfg configuration.Configuration, comps insapp.AppComponents) (insapp.AppComponent, error) {
	return NewAppCompartment(cfg.Ledger, comps), nil
}

func NewAppCompartment(_ configuration.Ledger, comps insapp.AppComponents) *insconveyor.AppCompartment {
	appDeps := injector.NewContainer(nil)

	// TODO move into AppComponents
	// appDeps.AddInterfaceDependency(&comps.BeatHistory)
	// appDeps.AddInterfaceDependency(&comps.AffinityHelper)
	// appDeps.AddInterfaceDependency(&comps.MessageSender)

	// convDeps := injector.NewContainer(nil)
	// convDeps.AddInterfaceDependency(&comps.MessageSender)

	appCmnt := insconveyor.NewAppCompartment("LMN", appDeps, nil,
		func(_ context.Context, _ injector.DependencyInjector, addFn managed.RegisterComponentFunc) {
			addFn(buildersvc.NewAdapterComponent(smadapter.AdapterExecutorConfig{}))
		},

		func(context.Context, injector.DependencyInjector) insconveyor.AppCompartmentSetup {
			f := NewEventFactory()
			return insconveyor.AppCompartmentSetup{
				ConveyorConfig: conveyor.PulseConveyorConfig{
					PulseSlotMigration: f.PostMigrate,
				},
				EventFactory: f,
			}
		})

	return appCmnt
}
