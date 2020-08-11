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
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/treesvc"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

// AppFactory is an entry point for ledger-core/server logic
func AppFactory(_ context.Context, cfg configuration.Configuration, comps insapp.AppComponents) (insapp.AppComponent, error) {
	return NewAppCompartment(cfg.Ledger, comps), nil
}

func NewAppCompartment(_ configuration.Ledger, comps insapp.AppComponents) *insconveyor.AppCompartment {
	appDeps := injector.NewDynamicContainer(nil)
	comps.AddInterfaceDependencies(appDeps)

	return insconveyor.NewAppCompartment("LMN", appDeps,

		func(ctx context.Context, _ injector.DependencyInjector, setup insconveyor.AppCompartmentSetup) insconveyor.AppCompartmentSetup {

			setup.Dependencies.AddInterfaceDependency(&comps.MessageSender)
			setup.Dependencies.AddInterfaceDependency(&comps.CryptoScheme)

			{
				var plashCatalog datawriter.PlashCataloger = datawriter.PlashCatalog{}
				var dropCatalog datawriter.DropCataloger = datawriter.DropCatalog{}
				var lineCatalog datawriter.LineCataloger = datawriter.LineCatalog{}

				setup.Dependencies.AddInterfaceDependency(&plashCatalog)
				setup.Dependencies.AddInterfaceDependency(&dropCatalog)
				setup.Dependencies.AddInterfaceDependency(&lineCatalog)

				var treeServiceImpl = treesvc.NewEmpty()
				var treeService treesvc.Service = treeServiceImpl

				setup.AddComponent(treeServiceImpl)
				setup.Dependencies.AddInterfaceDependency(&treeService)
			}

			setup.AddComponent(buildersvc.NewAdapterComponent(smadapter.Config{}, comps.CryptoScheme))

			f := NewEventFactory(ctx)
			setup.ConveyorConfig.PulseSlotMigration = f.PostMigrate
			setup.EventFactoryFn = f.InputEvent

			return setup
		})
}
