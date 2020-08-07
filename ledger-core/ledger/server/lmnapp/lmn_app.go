// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmnapp

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
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
	appDeps := injector.NewDynamicContainer(nil)
	comps.AddInterfaceDependencies(appDeps)

	return insconveyor.NewAppCompartment("LMN", appDeps,

		func(_ context.Context, _ injector.DependencyInjector, setup insconveyor.AppCompartmentSetup) insconveyor.AppCompartmentSetup {

			setup.Dependencies.AddInterfaceDependency(&comps.MessageSender)

			setup.AddComponent(buildersvc.NewAdapterComponent(smadapter.AdapterExecutorConfig{}))

			f := NewEventFactory()
			setup.ConveyorConfig.PulseSlotMigration = f.PostMigrate
			setup.EventFactoryFn = f.InputEvent

			return setup
		})
}